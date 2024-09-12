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

		log.Printf("CPU Usage: %v\n", containerStats.CPUStats.CPUUsage.TotalUsage)
		log.Printf("Memory Usage: %v / %v\n", containerStats.MemoryStats.Usage, containerStats.MemoryStats.Limit)
		log.Printf("Network I/O: Rx: %v, Tx: %v\n", containerStats.Networks["eth0"].RxBytes, containerStats.Networks["eth0"].TxBytes)
		log.Println()

		time.Sleep(1 * time.Second)
	}
}

// BRAINSTORM Points a copy of the volume to the same container like two to one volume

func Execute() {
	rootCmd.AddCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
