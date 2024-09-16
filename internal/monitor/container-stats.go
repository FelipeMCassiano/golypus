package monitor

import (
	"context"
	"encoding/json"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/sync/errgroup"
)

func monitorContainerStatus(ctx context.Context, containerId string, clt *client.Client) error {
	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		stats, err := clt.ContainerStats(gctx, containerId, true)
		if err != nil {
			if err == context.Canceled {
				log.Printf("Context cancelled while monitoring container %s", containerId)
				return nil
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
