package install

import (
	"autobutler/internal/install"
	"fmt"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Autobutler's system service",
		Long:  `The install command sets up Autobutler as a system service, allowing it to run in the background and start automatically on system boot.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Install Autobutler's system service")
			if err := install.Install(); err != nil {
				return fmt.Errorf("failed to install Autobutler as a system service; %w", err)
			}
			fmt.Println("Autobutler's system service was installed successfully.")
			return nil
		},
	}

	return cmd
}
