/*
Copyright © 2024 Felipe Cassiano felipecassianofmc@gmail.com
*/
package commands

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var rootCmd = &cobra.Command{
	Use:   "golypus",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		group, _ := errgroup.WithContext(context.Background())
		cntxt := &daemon.Context{
			PidFileName: "golypus.pid",
			PidFilePerm: 0644,
			LogFileName: "golypus.log",
			LogFilePerm: 0640,
		}
		d, err := cntxt.Reborn()
		if err != nil {
			return err
		}
		if d != nil {
			return nil
		}

		defer cntxt.Release()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		group.Go(listenDockerEvents)

		if err := group.Wait(); err != nil {
			return err
		}
		<-sigs

		return nil
	},
}

func listenDockerEvents() error {
	clt, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	messages, errs := clt.Events(context.Background(), events.ListOptions{})
	for {
		select {
		case msg := <-messages:
			if msg.Type == events.ContainerEventType && msg.Action == "create" {
				log.Printf("Container created: %s", msg.Actor.ID)
			}
		case err := <-errs:
			return err
		}
	}
}

func Execute() {
	rootCmd.AddCommand(CreateContainerCommand())
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
