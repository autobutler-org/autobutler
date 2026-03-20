package main

import (
	"fmt"
	"os"

	"github.com/autobutler-org/autobutler/cmd/autobutler/install"
	"github.com/autobutler-org/autobutler/cmd/autobutler/serve"
	"github.com/autobutler-org/autobutler/cmd/autobutler/smb"
	"github.com/autobutler-org/autobutler/cmd/autobutler/version"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "autobutler"}
	rootCmd.AddCommand(install.Cmd(), version.Cmd(), serve.Cmd(), smb.Cmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
