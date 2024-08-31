package subcommands

import (
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

func CreateContainerRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Create a container with a image",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			clt, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				return err
			}
			defer clt.Close()

			reader, err := clt.ImagePull(ctx, "postgres", image.PullOptions{})
			if err != nil {
				return err
			}

			defer reader.Close()

			if _, err := io.Copy(os.Stdout, reader); err != nil {
				return err
			}
			// BRAINSTORM: read a docker-compose.yml or .toml to configure that
			resp, err := clt.ContainerCreate(ctx, &container.Config{
				Image: "postgres",
				Tty:   false,
			}, nil, nil, nil, "")
			return nil
		},
	}

	return cmd
}
