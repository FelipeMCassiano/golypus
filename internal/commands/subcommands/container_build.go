package subcommands

// TODO: MAKE THIS FUNCTIONAL HA
// USE client.imageBuild

// func CreateContainerBuildCommand() *cobra.Command {
// 	var tagsFlag []string

// 	cmd := &cobra.Command{
// 		Use:   "build",
// 		Short: "Build a local Dockerfile",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			clt, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 			if err != nil {
// 				return err
// 			}
// 			if err := buildImageFromDockerFile(clt, tagsFlag, "Dockerfile"); err != nil {
// 				return err
// 			}

// 			return nil
// 		},
// 	}
// 	cmd.Flags().StringSliceVarP(&tagsFlag, "tags", "t", []string{}, "Defines tags in a image")

// 	return cmd
// }

// func buildImageFromDockerFile(client *client.Client, tags []string, dockerfile string) error {
// 	ctx := context.Background()

// 	buf := new(bytes.Buffer)
// 	tw := tar.NewWriter(buf)
// 	defer tw.Close()

// 	dockerFileReaer, err := os.Open(dockerfile)
// 	if err != nil {
// 		return err
// 	}

// 	readDockerFile, err := io.ReadAll(dockerFileReaer)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println(string(readDockerFile))

// 	tarHeader := &tar.Header{
// 		Name: dockerfile,
// 		Size: int64(len(readDockerFile)),
// 	}

// 	if err := tw.WriteHeader(tarHeader); err != nil {
// 		return err
// 	}

// 	dockerFileTarReader := bytes.NewReader(buf.Bytes())

// 	buildOptions := types.ImageBuildOptions{
// 		Context:    dockerFileTarReader,
// 		Dockerfile: dockerfile,
// 		Remove:     true,
// 		Tags:       tags,
// 	}

// 	imageBuildResponse, err := client.ImageBuild(ctx, dockerFileTarReader, buildOptions)
// 	if err != nil {
// 		return err
// 	}

// 	defer imageBuildResponse.Body.Close()

// 	if _, err := io.Copy(os.Stdout, imageBuildResponse.Body); err != nil {
// 		return err
// 	}

// 	return nil
// }
