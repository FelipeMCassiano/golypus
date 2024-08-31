/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "golypus",
	Short: "",
	Long:  ``,
}

func Execute() {
	rootCmd.AddCommand(CreateContainerCommand())
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
