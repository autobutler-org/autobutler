package cmd

import (
	"fmt"

	"github.com/exokomodo/exoflow/autobutler/backend/pkg/version"
	"github.com/spf13/cobra"
)

func Version() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information for Autobutler CLI",
		Long:  `The version command provides the current version of the Autobutler CLI and its components.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Here you would typically retrieve the version information from a variable or a configuration file.
			// For demonstration purposes, we'll use a hardcoded version string.
			version := version.GetVersion()
			fmt.Println(version.VersionString())
		},
	}

	return cmd
}
