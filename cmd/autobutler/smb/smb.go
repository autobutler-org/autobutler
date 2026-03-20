package smb

import (
	"fmt"

	"github.com/autobutler-org/autobutler/internal/smb"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "smb",
		Short: "Manage AutoButler's SMB (Samba) network share",
		Long:  "Configure and manage AutoButler as an SMB/CIFS network share for access from macOS Finder, Windows Explorer, and Linux file managers on the local network.",
	}

	cmd.AddCommand(setupCmd())
	cmd.AddCommand(statusCmd())
	return cmd
}

func setupCmd() *cobra.Command {
	var username string
	var password string

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install and configure the AutoButler SMB share (Linux only, requires sudo)",
		Long: `Installs Samba if needed, appends the AutoButler share block to /etc/samba/smb.conf,
sets the Samba password for a system user, and starts smbd.

Run with sudo: sudo autobutler smb setup --user <username> --password <password>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !smb.IsLinux() {
				return fmt.Errorf("smb setup is only supported on Linux")
			}
			if username == "" {
				return fmt.Errorf("--user is required")
			}
			if password == "" {
				return fmt.Errorf("--password is required")
			}
			return smb.Setup(username, password)
		},
	}

	cmd.Flags().StringVarP(&username, "user", "u", "", "System username for SMB access")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Samba password for the user")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current SMB setup status",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := smb.GetStatus()
			if err != nil {
				return err
			}

			if !s.Linux {
				fmt.Println("Platform: not Linux — SMB setup not supported")
				return nil
			}

			check := func(ok bool) string {
				if ok {
					return "✅"
				}
				return "❌"
			}

			fmt.Println("SMB Status:")
			fmt.Printf("  %s Samba installed\n", check(s.Installed))
			fmt.Printf("  %s Share configured (/etc/samba/smb.conf)\n", check(s.Configured))
			fmt.Printf("  %s smbd running\n", check(s.Running))
			if s.FilesDir != "" {
				fmt.Printf("  Files directory: %s\n", s.FilesDir)
			}
			return nil
		},
	}
}
