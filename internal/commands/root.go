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

	"github.com/FelipeMCassiano/golypus/internal/monitor"
	"github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var rootCmd = &cobra.Command{
	Use:   "golypus",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		group, ctx := errgroup.WithContext(context.Background())
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

		group.Go(func() error {
			select {
			case signal := <-sigs:
				log.Printf("Received signal: %s ... Shutdown", signal)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		monitorCtx, cancelMonitor := context.WithCancel(context.Background())
		defer cancelMonitor()

		group.Go(func() error {
			return monitor.ListenDockerEvents(monitorCtx)
		})

		if err := group.Wait(); err != nil {
			cancelMonitor()
			return err
		}

		select {}
	},
}

func Execute() {
	rootCmd.AddCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
