package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

func ConnectToDatabase() (*DatabaseSqlc, error) {
	var database DatabaseSqlc
	dataDir := storageutil.GetDataDir()
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dataFilePath := filepath.Join(dataDir, "autobutler.db")

	database.Db, err = sql.Open("sqlite", dataFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	sqlConn, err := database.Db.Conn(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %v", err)
	}
	database.Queries = New(sqlConn)

	if err := initSchema(&database); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %v", err)
	}

	return &database, nil
}

func ConnectToVaultDatabase(dbPath string) (*DatabaseSqlc, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create vault db directory: %v", err)
	}

	var database DatabaseSqlc
	var err error
	database.Db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open vault database: %v", err)
	}

	if err := InitVaultSchema(database.Db); err != nil {
		database.Db.Close()
		return nil, fmt.Errorf("failed to init vault schema: %v", err)
	}

	sqlConn, err := database.Db.Conn(context.Background())
	if err != nil {
		database.Db.Close()
		return nil, fmt.Errorf("failed to get vault db connection: %v", err)
	}
	database.Queries = New(sqlConn)

	return &database, nil
}

func ConnectToHealthDatabase() (*DatabaseRaw, error) {
	var database DatabaseRaw
	dataDir := storageutil.GetDataDir()
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		panic(fmt.Sprintf("failed to create data directory: %v", err))
	}

	healthFilePath := filepath.Join(dataDir, "autobutler.health.db")

	// Initialize health database for OTEL traces (no migrations needed)
	database.Db, err = sql.Open("sqlite", healthFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open health database: %v", err))
	}
	return &database, nil
}
