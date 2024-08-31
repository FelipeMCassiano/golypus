package commands

import (
	"github.com/FelipeMCassiano/golypus/internal/commands/subcommands"
	"github.com/spf13/cobra"
)

func CreateContainerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Manages docker containers",
	}

	cmd.AddCommand(subcommands.CreateContainerListCommand())
	return cmd
}
