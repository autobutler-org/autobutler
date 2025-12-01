package serve

import (
	"fmt"

	"autobutler/internal/server"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Autobutler server",
		Long:  `The serve command starts the Autobutler server, allowing you to interact with the Autobutler system through its API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Starting Autobutler server...")
			if err := server.StartServer(); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}
			return nil
		},
	}

	return cmd
}
