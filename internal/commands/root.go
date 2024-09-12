/*
Copyright © 2024 Felipe Cassiano felipecassianofmc@gmail.com
*/
package commands

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var rootCmd = &cobra.Command{
	Use:   "golypus",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		group, ctx := errgroup.WithContext(context.Background())
		cntxt := &daemon.Context{
			PidFileName: "golypus.pid",
			PidFilePerm: 0644,
			LogFileName: "golypus.log",
			LogFilePerm: 0640,
		}
		d, err := cntxt.Reborn()
		if err != nil {
			return err
		}
		if d != nil {
			return nil
		}

		defer cntxt.Release()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		group.Go(func() error {
			select {
			case signal := <-sigs:
				log.Printf("Recieved signal: %s ... Shutdown", signal)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

		group.Go(func() error {
			return listenDockerEvents(ctx)
		})

		if err := group.Wait(); err != nil {
			return err
		}
		select {}
	},
}

func listenDockerEvents(ctx context.Context) error {
	clt, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	group, _ := errgroup.WithContext(ctx)
	messages, errs := clt.Events(ctx, events.ListOptions{})
	for {
		select {
		case msg := <-messages:
			if msg.Type == events.ContainerEventType && msg.Action == "create" {
				group.Go(func() error {
					return monitorContainerStatus(ctx, msg.Actor.ID, clt)
				})
				log.Printf("Container created: %s\n", msg.Actor.ID)
			}
		case err := <-errs:
			return err
		}
	}
}

func monitorContainerStatus(ctx context.Context, containerId string, clt *client.Client) error {
	stats, err := clt.ContainerStats(ctx, containerId, true)
	if err != nil {
		return err
	}

	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)

	for {
		var containerStats container.StatsResponse
		err := decoder.Decode(&containerStats)
		if err != nil {
			return err
		}
		metrics := getMetrics(containerStats)
		autoScale(containerStats.ID, metrics)
		time.Sleep(1 * time.Second)
	}
}

type containerMetrics struct {
	MemUsed    float64
	MemAvail   float64
	CpuPerc    float64
	CpuMaxPerc float64
}

func getMetrics(containerStats container.StatsResponse) *containerMetrics {
	memUsed := containerStats.MemoryStats.Usage
	memAvail := containerStats.MemoryStats.Limit

	cpuUsage := containerStats.CPUStats.CPUUsage.TotalUsage - containerStats.PreCPUStats.CPUUsage.TotalUsage
	cpuSystem := containerStats.CPUStats.SystemUsage - containerStats.PreCPUStats.SystemUsage
	numCpus := containerStats.CPUStats.OnlineCPUs

	cpuPerc := (cpuUsage / cpuSystem) * uint64(numCpus) * 100
	cpuMaxPerc := numCpus * 100

	return &containerMetrics{
		MemUsed:    float64(memUsed),
		MemAvail:   float64(memAvail),
		CpuPerc:    float64(cpuPerc),
		CpuMaxPerc: float64(cpuMaxPerc),
	}
}

func autoScale(containerId string, metrics *containerMetrics) {
	if metrics.MemUsed == metrics.MemAvail*0.75 {
		// create a copy of the container
	}

	if metrics.CpuPerc == metrics.CpuMaxPerc*075 {
		// create a copy of the container
		// Use client.commit, i think
	}
}

// BRAINSTORM Points a copy of the volume to the same container like two to one volume
// Just saving it for later

func Execute() {
	rootCmd.AddCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
