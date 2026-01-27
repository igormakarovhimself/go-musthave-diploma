package storage

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"go-musthave-diploma/internal/logger"
)

func InitDB(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := RunMigrations(dsn); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func RunMigrations(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		logger.Log.Error("Failed to initialize migration tool", "error", err)
		return err
	}
	defer m.Close()

	current, _, _ := m.Version()
	logger.Log.Info("Current database schema version", "version", current)

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Log.Info("No migrations to apply")
			return nil
		}

		logger.Log.Error("Migration failed", "error", err)
		return err
	}

	newVersion, _, _ := m.Version()
	if newVersion > current {
		logger.Log.Info("Database schema migrated successfully",
			"from", current,
			"to", newVersion)
	} else {
		logger.Log.Info("Database schema is up to date")
	}

	return nil
}
