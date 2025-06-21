package cmd

import (
	"fmt"
	"os"

	"github.com/exokomodo/exoflow/autobutler/backend/internal/server"
	"github.com/spf13/cobra"
)

func Serve() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Autobutler server",
		Long:  `The serve command starts the Autobutler server, allowing you to interact with the Autobutler system through its API.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting Autobutler server...")
			if err := server.StartServer(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}
