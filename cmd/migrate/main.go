package main

import (
	"flag"
	"fmt"
	"os"

	"alertbot/internal/config"
	"alertbot/internal/migration"
	"alertbot/internal/repository"
	"alertbot/pkg/logger"

	"github.com/sirupsen/logrus"
)

func main() {
	// Parse command line flags
	drop := flag.Bool("drop", false, "Drop all tables before migrating")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	log := logger.New(cfg.Logger)

	// Connect to database
	db, err := repository.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create migrator
	migrator := migration.NewMigrator(db, log)

	// Drop tables if requested
	if *drop {
		log.Warn("Dropping all database tables")
		if err := migrator.DropAll(); err != nil {
			log.Fatalf("Failed to drop tables: %v", err)
		}
		log.Info("All tables dropped successfully")
	}

	// Run migrations
	log.Info("Starting database migration")
	if err := migrator.Migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Info("Database migration completed successfully")
	fmt.Println("✅ Migration completed successfully!")
}

