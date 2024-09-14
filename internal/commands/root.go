/*
Copyright © 2024 Felipe Cassiano felipecassianofmc@gmail.com
*/
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
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
				log.Printf("Received signal: %s ... Shutdown", signal)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		monitorCtx, cancelMonitor := context.WithCancel(context.Background())
		defer cancelMonitor()

		group.Go(func() error {
			return listenDockerEvents(monitorCtx)
		})

		if err := group.Wait(); err != nil {
			cancelMonitor()
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

	messages, errs := clt.Events(ctx, events.ListOptions{})
	group, gctx := errgroup.WithContext(ctx)

	for {
		select {
		case msg := <-messages:
			if msg.Type == events.ContainerEventType && msg.Action == "create" {
				go func(clt *client.Client, containerId string) {
					log.Printf("Container created: %s\n", msg.Actor.ID)
					<-time.After(30 * time.Second)
					group.Go(func() error {
						return monitorContainerStatus(gctx, msg.Actor.ID, clt)
					})
				}(clt, msg.Actor.ID)
			}
		case err := <-errs:
			if err != nil && err != context.Canceled {
				log.Printf("Error while listening to Docker events: %v", err)
				return err
			}
		case <-gctx.Done():
			// Handle context cancellation gracefully
			if gctx.Err() == context.Canceled {
				log.Println("Context cancelled, shutting down Docker event listener")
				return nil
			}
			return gctx.Err()
		}
	}
}

func monitorContainerStatus(ctx context.Context, containerId string, clt *client.Client) error {
	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		stats, err := clt.ContainerStats(gctx, containerId, true)
		if err != nil {
			if err == context.Canceled {
				log.Printf("Context cancelled while monitoring container %s", containerId)
				return nil // Gracefully handle cancellation
			}
			return err
		}

		defer stats.Body.Close()

		decoder := json.NewDecoder(stats.Body)

		var containerStats container.StatsResponse
		if err := decoder.Decode(&containerStats); err != nil {
			return err
		}

		metrics := getMetrics(containerStats)
		log.Printf("Metrics for container %s: %+v\n", containerId, metrics)

		if err := autoScale(ctx, containerId, metrics, clt); err != nil {
			return err
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		log.Printf("Error in monitorContainerStatus: %v", err)
		return err
	}

	return nil
}

type containerMetrics struct {
	MemUsed    uint64
	MemAvail   uint64
	CpuPerc    float64
	CpuMaxPerc float64
}

func getMetrics(containerStats container.StatsResponse) *containerMetrics {
	memUsed := containerStats.MemoryStats.Usage
	memAvail := containerStats.MemoryStats.Limit

	// Adjust CPU percentage calculation
	cpuDelta := float64(containerStats.CPUStats.CPUUsage.TotalUsage - containerStats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(containerStats.CPUStats.SystemUsage - containerStats.PreCPUStats.SystemUsage)
	numCpus := float64(containerStats.CPUStats.OnlineCPUs)
	cpuPerc := 0.0

	if cpuDelta > 0.0 && systemDelta > 0.0 {
		cpuPerc = (cpuDelta / systemDelta) * numCpus * 100
	}

	cpuMaxPerc := numCpus * 100

	return &containerMetrics{
		MemUsed:    memUsed,
		MemAvail:   memAvail,
		CpuPerc:    cpuPerc,
		CpuMaxPerc: cpuMaxPerc,
	}
}

var mutex sync.Mutex

func autoScale(ctx context.Context, containerId string, metrics *containerMetrics, clt *client.Client) error {
	mutex.Lock()
	defer mutex.Unlock()

	cooldown := 5 * time.Minute
	lastScaled := time.Now().Add(-cooldown)

	scaled := false

	for {
		// Log metrics and scaling decisions
		log.Printf("Memory used: %d, Memory available: %d (75%% threshold: %d)", metrics.MemUsed, metrics.MemAvail, (metrics.MemAvail*75)/100)
		log.Printf("CPU used: %.2f%%, Max CPU: %.2f%% (75%% threshold: %.2f%%)", metrics.CpuPerc, metrics.CpuMaxPerc, metrics.CpuMaxPerc*0.75)

		if metrics.MemUsed >= (metrics.MemAvail*75)/100 {
			s, lS, err := scaleWhenTheThresholdIsTriggered(scaled, lastScaled, cooldown, ctx, containerId, metrics, clt)
			if err != nil {
				return err
			}
			scaled = s
			lastScaled = lS
			continue
		}
		if metrics.CpuPerc >= metrics.CpuMaxPerc*0.75 {
			s, lS, err := scaleWhenTheThresholdIsTriggered(scaled, lastScaled, cooldown, ctx, containerId, metrics, clt)
			if err != nil {
				return err
			}
			scaled = s
			lastScaled = lS
			continue
		}

		// Wait for 1 minute before checking again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Minute):
		}
	}
}

// I didn't know a name better than this
func scaleWhenTheThresholdIsTriggered(scaled bool, lastScaled time.Time, cooldown time.Duration, ctx context.Context, containerId string, metrics *containerMetrics, clt *client.Client) (bool, time.Time, error) {
	if !scaled && time.Since(lastScaled) >= cooldown {
		created, err := performScaling(ctx, containerId, metrics, clt)
		if err != nil {
			return false, lastScaled, err
		}
		if created {
			scaled = true
			lastScaled = time.Now()
		}
	}

	return true, lastScaled, nil
}

func performScaling(ctx context.Context, containerId string, metrics *containerMetrics, clt *client.Client) (bool, error) {
	containerInfo, err := clt.ContainerInspect(ctx, containerId)
	if err != nil {
		return false, err
	}

	originalName := strings.TrimPrefix(containerInfo.Name, "/")
	copyContainerName := fmt.Sprintf("%s-copy-%s", originalName, uuid.NewString())

	hostConfig := containerInfo.HostConfig

	for port, bindings := range hostConfig.PortBindings {
		for i := range bindings {
			bindings[i].HostPort = "" // Docker will assign an available port
		}

		hostConfig.PortBindings[port] = bindings
	}

	resp, err := clt.ContainerCreate(ctx, containerInfo.Config, hostConfig, &network.NetworkingConfig{
		EndpointsConfig: containerInfo.NetworkSettings.Networks,
	}, nil, copyContainerName)
	if err != nil {
		return false, err
	}

	if err := clt.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return false, err
	}

	log.Printf("Scaled container %s to %s\n", containerId, resp.ID)
	return true, nil
}

func gracefulShutdown() error {
	return nil
}

func Execute() {
	rootCmd.AddCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
