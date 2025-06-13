package cmd

import (
	"fmt"
	"os"
	"context"
	"os/signal"
	"syscall"
	"time"
	
	"dotai-go-backend/internal/routes"
	"dotai-go-backend/internal/database"

	"github.com/spf13/cobra"
)

func Serve() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Dotai server",
		Long:  `The serve command starts the Dotai server, allowing you to interact with the Dotai system through its API.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting Dotai server...")

			ctx := context.Background()

			dbConfig := database.LoadConfig()
			db, err := database.NewConnection(ctx, dbConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
				os.Exit(1)
			}
			defer db.Close()

			fmt.Println("Database connection successful")

			if err := startServer(ctx, db); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}

func startServer(ctx context.Context, db *database.DB) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- routes.StartServer(ctx, db)
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case <-interrupt:
		fmt.Println("Shutting down server...")
		cancel()	

		time.Sleep(2 * time.Second)
		return nil
	}

}