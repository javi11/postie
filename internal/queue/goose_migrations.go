package queue

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// GooseMigrationRunner handles database migrations using goose
type GooseMigrationRunner struct {
	db *sql.DB
}

// NewGooseMigrationRunner creates a new goose migration runner
func NewGooseMigrationRunner(db *sql.DB) *GooseMigrationRunner {
	return &GooseMigrationRunner{db: db}
}

// SetupGoose initializes goose with embedded migrations
func (gmr *GooseMigrationRunner) SetupGoose() error {
	// Set goose to use embedded migrations
	goose.SetBaseFS(embedMigrations)

	// Set migration directory within the embedded filesystem
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	return nil
}

// MigrateUp runs all pending migrations
func (gmr *GooseMigrationRunner) MigrateUp() error {
	if err := gmr.SetupGoose(); err != nil {
		return err
	}

	slog.Info("Running database migrations")

	if err := goose.Up(gmr.db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

// MigrateDown rolls back the last migration
func (gmr *GooseMigrationRunner) MigrateDown() error {
	if err := gmr.SetupGoose(); err != nil {
		return err
	}

	slog.Info("Rolling back last migration")

	if err := goose.Down(gmr.db, "migrations"); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	slog.Info("Migration rollback completed successfully")
	return nil
}

// MigrateTo migrates to a specific version
func (gmr *GooseMigrationRunner) MigrateTo(version int64) error {
	if err := gmr.SetupGoose(); err != nil {
		return err
	}

	slog.Info("Migrating to specific version", "version", version)

	if err := goose.UpTo(gmr.db, "migrations", version); err != nil {
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}

	slog.Info("Migration to version completed successfully", "version", version)
	return nil
}

// GetStatus returns the current migration status
func (gmr *GooseMigrationRunner) GetStatus() (*GooseMigrationStatus, error) {
	if err := gmr.SetupGoose(); err != nil {
		return nil, err
	}

	// Get current version
	currentVersion, err := goose.GetDBVersion(gmr.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	return &GooseMigrationStatus{
		CurrentVersion: currentVersion,
	}, nil
}

// Reset drops all tables and re-runs all migrations
func (gmr *GooseMigrationRunner) Reset() error {
	if err := gmr.SetupGoose(); err != nil {
		return err
	}

	slog.Info("Resetting database - dropping all tables and re-running migrations")

	if err := goose.Reset(gmr.db, "migrations"); err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	slog.Info("Database reset completed successfully")
	return nil
}

// IsLegacyDatabase checks if the database exists but doesn't have goose migrations table
func (gmr *GooseMigrationRunner) IsLegacyDatabase() (bool, error) {
	// Check if goqite table exists (indicating an existing database)
	var tableExists bool
	err := gmr.db.QueryRow(`
		SELECT EXISTS(
			SELECT name FROM sqlite_master 
			WHERE type='table' AND name='goqite'
		)
	`).Scan(&tableExists)

	if err != nil {
		return false, fmt.Errorf("failed to check for goqite table: %w", err)
	}

	if !tableExists {
		// No database exists yet
		return false, nil
	}

	// Check if goose version table exists
	var gooseTableExists bool
	err = gmr.db.QueryRow(`
		SELECT EXISTS(
			SELECT name FROM sqlite_master 
			WHERE type='table' AND name='goose_db_version'
		)
	`).Scan(&gooseTableExists)

	if err != nil {
		return false, fmt.Errorf("failed to check for goose version table: %w", err)
	}

	// If goqite exists but goose table doesn't, it's a legacy database
	return tableExists && !gooseTableExists, nil
}

// RecreateDatabase drops all existing tables and recreates them using goose migrations
func (gmr *GooseMigrationRunner) RecreateDatabase() error {
	slog.Info("Recreating database - dropping all existing tables")

	// Get all table names
	rows, err := gmr.db.Query(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
	`)
	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Drop all tables in a transaction
	tx, err := gmr.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Disable foreign key constraints during drop
	if _, err := tx.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}

	// Drop all tables
	for _, table := range tables {
		slog.Debug("Dropping table", "table", table)
		if _, err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Re-enable foreign key constraints
	if _, err := tx.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to re-enable foreign keys: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit table drops: %w", err)
	}

	slog.Info("All tables dropped successfully")

	// Now run goose migrations to recreate everything
	return gmr.MigrateUp()
}

// EnsureMigrationCompatibility checks for legacy database and recreates if needed
func (gmr *GooseMigrationRunner) EnsureMigrationCompatibility() error {
	isLegacy, err := gmr.IsLegacyDatabase()
	if err != nil {
		return fmt.Errorf("failed to check for legacy database: %w", err)
	}

	if isLegacy {
		slog.Warn("Legacy database detected - recreating with goose migration system")
		return gmr.RecreateDatabase()
	}

	// Normal migration flow
	return gmr.MigrateUp()
}

// GooseMigrationStatus represents the current migration state using goose
type GooseMigrationStatus struct {
	CurrentVersion int64 `json:"currentVersion"`
}
