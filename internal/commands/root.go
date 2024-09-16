/*
Copyright © 2024 Felipe Cassiano felipecassianofmc@gmail.com
*/
package commands

import (
	"context"
	"log"
	"os"
	"syscall"

	"github.com/FelipeMCassiano/golypus/internal/containers/monitor"
	"github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func CreateRootCommand() *cobra.Command {
	var signalRecieved string
	rootCmd := &cobra.Command{
		Use:     "golypus",
		Short:   "Monitor docker containers and scale them if necessary",
		Version: "0.0.1",
		RunE: func(cmd *cobra.Command, args []string) error {
			group, ctx := errgroup.WithContext(context.Background())

			daemon.AddCommand(daemon.StringFlag(&signalRecieved, "quit"), syscall.SIGQUIT, termHandler)
			daemon.AddCommand(daemon.StringFlag(&signalRecieved, "stop"), syscall.SIGTERM, termHandler)

			cntxt := &daemon.Context{
				PidFileName: "golypus.pid",
				PidFilePerm: 0644,
				LogFileName: "golypus.log",
				LogFilePerm: 0640,
			}

			if len(daemon.ActiveFlags()) > 0 {
				d, err := cntxt.Search()
				if err != nil {
					return err
				}

				err = daemon.SendCommands(d)
				if err != nil {
					return err
				}
				return nil
			}

			d, err := cntxt.Reborn()
			if err != nil {
				return err
			}
			if d != nil {
				return nil
			}

			defer cntxt.Release()

			monitorCtx, cancelMonitor := context.WithCancel(context.Background())
			defer cancelMonitor()

			group.Go(func() error {
				return monitor.ListenDockerEvents(monitorCtx)
			})

			if err := group.Wait(); err != nil {
				cancelMonitor()
				return err
			}

			err = daemon.ServeSignals()
			if err != nil {
				return err
			}

			<-ctx.Done()
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&signalRecieved, "signal", "s", " ", "Send signal ")
	return rootCmd
}

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func termHandler(sig os.Signal) error {
	log.Println("terminating...")
	stop <- struct{}{}

	if sig == syscall.SIGQUIT {
		<-done
	}
	return daemon.ErrStop
}

func Execute() {
	rootCmd := CreateRootCommand()
	rootCmd.AddCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
