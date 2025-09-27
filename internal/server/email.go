package server

import (
	"autobutler/pkg/util"
	"fmt"

	"github.com/autobutler-ai/go-guerrilla"
	"github.com/autobutler-ai/go-guerrilla/backends"
)

func serveEmail() error {
	config := &guerrilla.AppConfig{
		LogFile: "stdout",
		BackendConfig: backends.BackendConfig{
			"gw_save_timeout":     "1s",
			"gw_val_rcpt_timeout": "1s",
			"log_received_mails":  true,
			"mail_table":          "emails",
			"primary_mail_host":   "example.com",
			"save_process":        "sql",
			"save_workers_size":   1,
			"sql_driver":          "sqlite",
			"sql_dsn":             util.GetDatabasePath(),
		},
	}
	daemon := guerrilla.Daemon{
		Config: config,
	}
	err := daemon.Start()
	if err != nil {
		fmt.Printf("Error starting email server: %s", err)
		return err
	}
	fmt.Println("Server Started!")
	return nil
}
