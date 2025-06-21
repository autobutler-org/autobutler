package cmd

import (
	"fmt"
	"os"

	"github.com/exokomodo/exoflow/autobutler/backend/internal/update"
	"github.com/spf13/cobra"
)

func Update() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Autobutler to a version",
		Long:  `The update command updates Autobutler to the specified version. It downloads the latest binary and replaces the current executable.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Println("Please specify a version to update to.")
				return
			}
			version := args[0]

			fmt.Printf("Updating Autobutler to version %s...\n", version)
			if err := update.Update(version); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating Autobutler: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Update successful, Autobutler will restart.")
		},
	}

	return cmd
}
