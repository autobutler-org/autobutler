package main

import (
	"fmt"
	"os"

	"autobutler/cmd/autobutler/install"
	"autobutler/cmd/autobutler/serve"
	"autobutler/cmd/autobutler/version"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "autobutler"}
	rootCmd.AddCommand(install.Cmd(), version.Cmd(), serve.Cmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
