package cmd

import (
	"autobutler/internal/install"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Install() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Autobutler's system service",
		Long:  `The install command sets up Autobutler as a system service, allowing it to run in the background and start automatically on system boot.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Install Autobutler's system service")
			if err := install.Install(); err != nil {
				fmt.Fprintf(os.Stderr, "Error install Autobutler as a system service: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Autobutler's system service was installed successfully.")
		},
	}

	return cmd
}
