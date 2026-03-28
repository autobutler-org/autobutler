package serve

import (
	"fmt"
	"os"

	"github.com/autobutler-org/autobutler/internal/server"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Autobutler server",
		Long:  `The serve command starts the Autobutler server, allowing you to interact with the Autobutler system through its API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Getenv("AUTOBUTLER_BRANCH") == "" {
				if branch := settingsutil.GetActiveBranch(); branch != "" {
					os.Setenv("AUTOBUTLER_BRANCH", branch)
				}
			}
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
