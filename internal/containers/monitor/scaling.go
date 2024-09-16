package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func autoScale(ctx context.Context, containerId string, metrics *containerMetrics, clt *client.Client) error {
	cooldown := 5 * time.Minute
	lastScaled := time.Now().Add(-cooldown)

	scaled := false

	for {

		// log.Printf("Memory used: %d, Memory available: %d (75%% threshold: %d)", metrics.MemUsed, metrics.MemAvail, (metrics.MemAvail*75)/100)
		// log.Printf("CPU used: %.2f%%, Max CPU: %.2f%% (75%% threshold: %.2f%%)", metrics.CpuPerc, metrics.CpuMaxPerc, metrics.CpuMaxPerc*0.75)

		if metrics.MemUsed >= (metrics.MemAvail*75)/100 {
			s, lS, err := scaleWhenTheThresholdIsTriggered(scaled, lastScaled, cooldown, ctx, containerId, clt)
			if err != nil {
				return err
			}
			scaled = s
			lastScaled = lS
			continue
		}
		if metrics.CpuPerc >= metrics.CpuMaxPerc*0.75 {
			s, lS, err := scaleWhenTheThresholdIsTriggered(scaled, lastScaled, cooldown, ctx, containerId, clt)
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
func scaleWhenTheThresholdIsTriggered(scaled bool, lastScaled time.Time, cooldown time.Duration, ctx context.Context, containerId string, clt *client.Client) (bool, time.Time, error) {
	if !scaled && time.Since(lastScaled) >= cooldown {
		created, err := performScaling(ctx, containerId, clt)
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

const COPY_NAME_SUFFIX = "copy"

func performScaling(ctx context.Context, containerId string, clt *client.Client) (bool, error) {
	containerInfo, err := clt.ContainerInspect(ctx, containerId)
	if err != nil {
		return false, err
	}

	originalName := strings.TrimPrefix(containerInfo.Name, "/")

	// cannot have the copy of the copy
	if strings.HasSuffix(originalName, COPY_NAME_SUFFIX) {
		return true, nil
	}
	copyContainerName := fmt.Sprintf("%s-%s", originalName, COPY_NAME_SUFFIX)

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

	return true, nil
}
