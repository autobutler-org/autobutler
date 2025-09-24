package db

import (
	"autobutler/pkg/util"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

var DatabaseQueries *Queries

func init() {
	dataDir := util.GetDataDir()
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		panic(fmt.Sprintf("failed to create data directory: %v", err))
	}

	dataFilePath := filepath.Join(dataDir, "autobutler.db")

	Instance.Db, err = sql.Open("sqlite", dataFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open database: %v", err))
	}
	sqlConn, err := Instance.Db.Conn(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database connection: %v", err))
	}
	DatabaseQueries = New(sqlConn)

	if err := initSchema(); err != nil {
		panic(fmt.Sprintf("failed to initialize database schema: %v", err))
	}
	if err := seedData(); err != nil {
		panic(fmt.Sprintf("failed to seed database: %v", err))
	}
}
