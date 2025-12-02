package serve

import (
	"fmt"

	"autobutler/internal/server"
	"autobutler/pkg/util/deputil"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Autobutler server",
		Long:  `The serve command starts the Autobutler server, allowing you to interact with the Autobutler system through its serverutil.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Starting Autobutler server...")
			deps, err := deputil.DefaultDependencies()
			if err != nil {
				return fmt.Errorf("failed to initialize dependencies: %w", err)
			}
			if err := server.StartServer(deps); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}
			return nil
		},
	}

	return cmd
}
