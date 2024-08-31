package subcommands

import (
	"github.com/spf13/cobra"
)

// TODO: Make this command functional cause, the container list api call is not working (idk)
func CreateContainerListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List all containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}
