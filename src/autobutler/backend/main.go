package main

import (
	"fmt"
	"os"

	"github.com/exokomodo/exoflow/autobutler/backend/cmd"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "autobutler"}
	rootCmd.AddCommand(cmd.Serve(), cmd.Chat(), cmd.Version(), cmd.Update(), cmd.Database())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
