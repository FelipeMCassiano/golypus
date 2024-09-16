package monitor

import (
	"context"
	"log"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"golang.org/x/sync/errgroup"
)

const MAX_MONITORS = 10

func ListenDockerEvents(ctx context.Context) error {
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
				log.Printf("Container created: %s\n", msg.Actor.ID)
				go func(ctx context.Context, containerId string, clt *client.Client) {
					<-time.After(30 * time.Second)
					group.Go(func() error {
						return monitorContainerStats(ctx, containerId, clt)
					})
				}(gctx, msg.Actor.ID, clt)
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
