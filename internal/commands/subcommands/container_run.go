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
	var imageFlag string
	var containerNameFlag string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Create a container with a image",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				return nil
			}
			defer cli.Close()

			// I thought about reading a docker-compose.yml file for this
			reader, err := cli.ImagePull(ctx, imageFlag, image.PullOptions{})
			if err != nil {
				return nil
			}

			defer reader.Close()
			if _, err := io.Copy(os.Stdout, reader); err != nil {
				return err
			}

			// ADD additicional configuration when creating == remove these nils
			resp, err := cli.ContainerCreate(ctx, &container.Config{
				Image: imageFlag,
				Tty:   false,
			}, nil, nil, nil, containerNameFlag)
			if err != nil {
				return err
			}

			if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&imageFlag, "image", "i", "", "Define the image of the container")
	cmd.Flags().StringVarP(&containerNameFlag, "container-name", "n", "", "Define the name of the container")
	cmd.MarkFlagRequired("image")

	return cmd
}

// Just a ideia how to make it

// func runContainer(client *client.Client, imagename string, containername string, port string, inputEnv []string) error {
// 	newport, err := nat.NewPort("tcp", port)
// 	if err != nil {
// 		return err
// 	}

// 	hostConfig := &container.HostConfig{
// 		PortBindings: nat.PortMap{
// 			newport: []nat.PortBinding{
// 				{
// 					HostIP:   "0.0.0.0",
// 					HostPort: port,
// 				},
// 			},
// 		},
// 		RestartPolicy: container.RestartPolicy{
// 			Name: "always",
// 		},

// 		LogConfig: container.LogConfig{
// 			Type:   "json-file",
// 			Config: map[string]string{},
// 		},
// 	}

// 	networkConfig := &network.NetworkingConfig{
// 		EndpointsConfig: map[string]*network.EndpointSettings{},
// 	}

// 	gatewayConfig := &network.EndpointSettings{
// 		Gateway: "gateway-name",
// 	}

// 	networkConfig.EndpointsConfig["bridge"] = gatewayConfig

// 	exposedPorts := map[nat.Port]struct{}{
// 		newport: {},
// 	}

// 	config := &container.Config{
// 		Image:        imagename,
// 		Env:          inputEnv,
// 		ExposedPorts: exposedPorts,
// 		Hostname:     fmt.Sprintf("%s-hostnameexample", imagename),
// 	}

// 	cont, err := client.ContainerCreate(context.Background(), config, hostConfig, networkConfig, nil, containername)
// 	if err != nil {
// 		return err
// 	}

// 	if err := client.ContainerStart(context.Background(), cont.ID, container.StartOptions{}); err != nil {
// 		return err
// 	}

// 	log.Printf("Container created id %s", cont.ID)

// 	return nil
// }
