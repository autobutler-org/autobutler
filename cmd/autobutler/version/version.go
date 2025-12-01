package version

import (
	"fmt"

	"autobutler/pkg/util/versionutil"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information for Autobutler CLI",
		Long:  `The version command provides the current version of the Autobutler CLI and its components.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := versionutil.GetVersion()
			fmt.Println(version.VersionString())
			return nil
		},
	}

	return cmd
}
