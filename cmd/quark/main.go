package main

import (
	"fmt"
	"os"

	"github.com/autobutler-org/quark/cmd/quark/install"
	"github.com/autobutler-org/quark/cmd/quark/serve"
	"github.com/autobutler-org/quark/cmd/quark/smb"
	"github.com/autobutler-org/quark/cmd/quark/version"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "quark"}
	rootCmd.AddCommand(install.Cmd(), version.Cmd(), serve.Cmd(), smb.Cmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
