package serve

import (
	"fmt"
	"os"

	"github.com/autobutler-org/autobutler/internal/server"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	var insecure bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Autobutler server",
		Long:  `The serve command starts the Autobutler server, allowing you to interact with the Autobutler system through its API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// AUTOBUTLER_INSECURE=true is equivalent to --insecure, useful for
			// Makefile targets and CI without modifying command-line args.
			if !insecure && os.Getenv("AUTOBUTLER_INSECURE") == "true" {
				insecure = true
			}
			fmt.Println("Starting Autobutler server...")
			deps, err := deputil.DefaultDependencies()
			if err != nil {
				return fmt.Errorf("failed to initialize dependencies: %w", err)
			}
			if err := server.StartServer(deps, server.StartOptions{Insecure: insecure}); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&insecure, "insecure", false, "Disable TLS and serve over plain HTTP (for local development only). Also enabled via AUTOBUTLER_INSECURE=true env var.")

	return cmd
}
