package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/FelipeMCassiano/golypus/internal/loadbalancer"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
)

func autoScale(ctx context.Context, containerId string, metrics *containerMetrics, clt *client.Client) error {
	cooldown := 5 * time.Minute
	lastScaled := time.Now().Add(-cooldown)
	scaled := false

	url := "ws://localhost:4444/loadbalancer/create"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	for {
		if metrics.MemUsed >= (metrics.MemAvail*75)/100 || metrics.CpuPerc >= metrics.CpuMaxPerc*0.75 {
			if !scaled && time.Since(lastScaled) >= cooldown {
				created, req, err := performScaling(ctx, containerId, clt)
				if err != nil {
					return err
				}
				if created {
					scaled = true
					lastScaled = time.Now()

					encodendMessage, err := json.Marshal(req)
					if err != nil {
						return err
					}

					err = conn.WriteMessage(websocket.TextMessage, encodendMessage)
					if err != nil {
						return err
					}

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

func performScaling(ctx context.Context, containerId string, clt *client.Client) (bool, *loadbalancer.CreateLoadbalancerReq, error) {
	containerInfo, err := clt.ContainerInspect(ctx, containerId)
	if err != nil {
		return false, nil, err
	}
	defer clt.ContainerRemove(ctx, containerInfo.ID, container.RemoveOptions{})

	originalName := strings.TrimPrefix(containerInfo.Name, "/")

	// cannot have the copy of the copy
	if strings.HasSuffix(originalName, COPY_NAME_SUFFIX) {
		return true, nil, nil
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
		return false, nil, nil
	}

	copyContainerName := fmt.Sprintf("%s-%s", originalName, COPY_NAME_SUFFIX)

	copyId, err := createContainerAndStart(ctx, containerInfo, copyContainerName, clt)
	if err != nil {
		return false, nil, nil
	}

	containerPort, err := getPortsOfContainer(ctx, originalId, clt)
	if err != nil {
		return false, nil, nil
	}

	copyPort, err := getPortsOfContainer(ctx, copyId, clt)
	if err != nil {
		return false, nil, nil
	}

	ports := append(loadbalancer.ContainerPorts{}, append(containerPort, copyPort...)...)

	var originalPort string

	for _, portsBindings := range containerInfo.NetworkSettings.Ports {
		if len(portsBindings) > 0 {
			originalPort = portsBindings[0].HostPort
		}
	}

	createReq := &loadbalancer.CreateLoadbalancerReq{
		LoadBalancerPort: originalPort,
		Ports:            ports,
	}

	return true, createReq, nil
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
			hostPorts = append(hostPorts, string(containerPort))
		}
	}

	return hostPorts, nil
}
