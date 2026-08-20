package db

import (
	"fmt"
	"log"
	"os"

	"github.com/deployerai/deployer/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// InitDB initializes the database connection and runs auto-migrations.
func InitDB(dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}

	if dsn != "" {
		log.Println("Connecting to PostgreSQL database...")
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	} else {
		log.Println("DATABASE_URL not set. Falling back to local SQLite database...")
		db, err = gorm.Open(sqlite.Open("deployer.db"), &gorm.Config{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	log.Println("Successfully connected to the database. Running migrations...")

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.Project{},
		&models.LLMCredential{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	log.Println("Database migration completed.")
	return db, nil
}
