package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	loadbalancer "github.com/FelipeMCassiano/golypus/internal/containers/load-balancer"
	"github.com/docker/docker/api/types"
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

		if metrics.MemUsed >= (metrics.MemAvail*75)/100 || metrics.CpuPerc >= metrics.CpuMaxPerc*0.75 {
			if !scaled && time.Since(lastScaled) >= cooldown {
				created, originalPort, containerPorts, err := performScaling(ctx, containerId, clt)
				if err != nil {
					return err
				}
				if created {
					scaled = true
					lastScaled = time.Now()

					loadbalancer.NewLoadBalancer(originalPort, containerPorts)

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
	}
}

const COPY_NAME_SUFFIX = "copy"

func performScaling(ctx context.Context, containerId string, clt *client.Client) (bool, string, []loadbalancer.ContainerPorts, error) {
	containerInfo, err := clt.ContainerInspect(ctx, containerId)
	if err != nil {
		return false, "", nil, err
	}
	defer clt.ContainerRemove(ctx, containerInfo.ID, container.RemoveOptions{})

	originalName := strings.TrimPrefix(containerInfo.Name, "/")

	// cannot have the copy of the copy
	if strings.HasSuffix(originalName, COPY_NAME_SUFFIX) {
		return true, "", nil, nil
	}

	hostConfig := containerInfo.HostConfig

	for port, bindings := range hostConfig.PortBindings {
		for i := range bindings {
			bindings[i].HostPort = "" // Docker will assign an available port
		}

		hostConfig.PortBindings[port] = bindings
	}

	originalId, err := createContainerAndStart(ctx, containerInfo, originalName, clt)
	if err != nil {
		return false, "", nil, nil
	}

	copyContainerName := fmt.Sprintf("%s-%s", originalName, COPY_NAME_SUFFIX)

	copyId, err := createContainerAndStart(ctx, containerInfo, copyContainerName, clt)
	if err != nil {
		return false, "", nil, nil
	}

	containerPort, err := getPortsOfContainer(ctx, originalId, clt)
	if err != nil {
		return false, "", nil, nil
	}

	copyPort, err := getPortsOfContainer(ctx, copyId, clt)
	if err != nil {
		return false, "", nil, nil
	}

	ports := []loadbalancer.ContainerPorts{containerPort, copyPort}

	var originalPort string

	for _, portsBindings := range containerInfo.NetworkSettings.Ports {
		if len(portsBindings) > 0 {
			originalPort = portsBindings[0].HostPort
		}
	}

	return true, originalPort, ports, nil
}

func createContainerAndStart(ctx context.Context, containerInfo types.ContainerJSON, name string, clt *client.Client) (string, error) {
	resp, err := clt.ContainerCreate(ctx, containerInfo.Config, containerInfo.HostConfig, &network.NetworkingConfig{EndpointsConfig: containerInfo.NetworkSettings.Networks}, nil, name)
	if err != nil {
		return "", err
	}

	if err := clt.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func getPortsOfContainer(ctx context.Context, containerId string, clt *client.Client) (loadbalancer.ContainerPorts, error) {
	containerInfo, err := clt.ContainerInspect(ctx, containerId)
	if err != nil {
		return nil, err
	}
	hostPorts := loadbalancer.ContainerPorts{}
	for containerPort, portsBindings := range containerInfo.NetworkSettings.Ports {
		if len(portsBindings) > 0 {
			hostPorts[string(containerPort)] = portsBindings[0].HostPort
		}
	}
	return hostPorts, nil
}
