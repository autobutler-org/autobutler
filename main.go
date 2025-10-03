package main

import (
	"fmt"
	"os"

	"autobutler/cmd"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "autobutler"}
	rootCmd.AddCommand(cmd.Serve(), cmd.Version(), cmd.Install(), cmd.Update())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
