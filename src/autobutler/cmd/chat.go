package cmd

import (
	"fmt"

	"autobutler/internal/llm"

	"github.com/spf13/cobra"
)

func Chat() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Interact with the Autobutler chat system",
		Long:  `The chat command allows you to interact with the Autobutler chat system, enabling you to send and receive messages.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Please provide a message to send.")
				return
			}
			message := args[0]
			response, err := llm.DoChat(message)
			if err != nil {
				fmt.Printf("Error sending message: %v\n", err)
				return
			}
			fmt.Println(response)
		},
	}

	return cmd
}
