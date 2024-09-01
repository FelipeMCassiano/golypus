package subcommands

import (
	"context"
	"fmt"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

func CreateContainerListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List all containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				return err
			}
			defer cli.Close()

			containers, err := cli.ContainerList(ctx, containertypes.ListOptions{})
			if err != nil {
				return err
			}
			for _, container := range containers {
				fmt.Println(container)
			}
			return nil
		},
	}
	return cmd
}
