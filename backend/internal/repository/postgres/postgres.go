package postgres

import (
	"chat-app/internal/config"
	"chat-app/internal/logger"
	"chat-app/internal/repository/db"
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// Ensure PostgresDB implements db.Database interface
var _ db.Database = (*PostgresDB)(nil)

// PostgresDB implements the db.Database interface
type PostgresDB struct {
	conn *sql.DB
}

// NewPostgresDB creates a new PostgresDB instance with a new connection
func NewPostgresDB(dbConfig config.DatabaseConfig) (*PostgresDB, error) {
	dsn := dbConfig.GetDSN()
	logger.Log.WithField("dsn", dsn).Info("Connecting to PostgreSQL")

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Test the connection
	if err = conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	logger.Log.Info("Successfully connected to PostgreSQL")

	db := &PostgresDB{conn: conn}

	// Run migrations
	if err = db.RunMigrations(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	logger.Log.Info("Migrations completed successfully")

	return db, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// GetDB returns the underlying database connection
// This is used by methods that need direct access to *sql.DB
func (p *PostgresDB) GetDB() *sql.DB {
	return p.conn
}

// RunMigrations runs database migrations using golang-migrate
func (p *PostgresDB) RunMigrations() error {
	driver, err := postgres.WithInstance(p.conn, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error creating migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("error creating migration instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("error running migrations: %w", err)
	}

	logger.Log.Info("Database migrations applied successfully")
	return nil
}
