package cmd

import (
	"fmt"

	"autobutler/internal/db"

	"github.com/spf13/cobra"
)

func Database() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Example command to interact with the database",
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.ExampleDatabase(); err != nil {
				fmt.Printf("Error interacting with the database: %v\n", err)
				return
			}
		},
	}

	return cmd
}
