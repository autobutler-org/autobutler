package main

import (
	"fmt"
	"os"

	"autobutler/cmd"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "autobutler"}
	rootCmd.AddCommand(cmd.Serve(), cmd.Chat(), cmd.Version(), cmd.Update(), cmd.Database(), cmd.Mcp())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
