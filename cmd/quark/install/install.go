package install

import (
	"fmt"

	"github.com/autobutler-org/quark/internal/install"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Quark's system service",
		Long:  `The install command sets up Quark as a system service, allowing it to run in the background and start automatically on system boot.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Install Quark's system service")
			if err := install.Install(); err != nil {
				return fmt.Errorf("failed to install Quark as a system service; %w", err)
			}
			fmt.Println("Quark's system service was installed successfully.")
			return nil
		},
	}

	return cmd
}
