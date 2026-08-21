package version

import (
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/versionutil"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information for Quark CLI",
		Long:  `The version command provides the current version of the Quark CLI and its components.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := versionutil.GetVersion()
			fmt.Println(version.VersionString())
			return nil
		},
	}

	return cmd
}
