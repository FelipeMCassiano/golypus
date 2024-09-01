package subcommands

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

func CreateStopContainerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stops the container based on its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}

func stopContainer(client *client.Client, containerName string) error {
	ctx := context.Background()
	if err := client.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		return err
	}
	return nil
}
