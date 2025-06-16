package database

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"log"
)

type DBManager struct {
	DB   *sql.DB
	path string
}

func NewDBManager(dbPath string) *DBManager {
	return &DBManager{
		path: dbPath,
	}
}

func (dm *DBManager) Connect() error {
	log.Printf("Attempting to connect to database at: %s", dm.path)
	db, err := sql.Open("sqlite3", dm.path)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	dm.DB = db
	if err = dm.DB.Ping(); err != nil {
		dm.DB.Close() // Close if ping fails
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to database.")
	return nil
}
func (dm *DBManager) Close() error {
	if dm.DB != nil {
		log.Println("Closing database connection.")
		return dm.DB.Close()
	}
	return nil
}

func (dm *DBManager) ApplyMigrations() error {
	if dm.DB == nil {
		return errors.New("database connection is not established, call Connect() first") // Use errors package from "errors"
	}

	log.Println("Applying database migrations...")
	_, err := dm.DB.Exec(createSchemaSQL) // createSchemaSQL is still in migrations.go
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	log.Println("Database migrations applied successfully.")
	_, err = dm.DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	log.Println("Foreign keys enabled for connection.")

	return nil
}
