package db

import (
	"chat-app/internal/config"
	"chat-app/internal/logger"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresDB implements the Database interface
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

	// Create tables
	if err = db.createTables(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("error creating tables: %w", err)
	}

	logger.Log.Info("Tables created/verified")

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

// createTables creates the necessary database tables
func (p *PostgresDB) createTables() error {
	db := p.conn

	// Create users table
	usersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255),
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`

	if _, err := db.Exec(usersTableSQL); err != nil {
		return fmt.Errorf("error creating users table: %w", err)
	}

	// Create conversations table
	conversationsTableSQL := `
	CREATE TABLE IF NOT EXISTS conversations (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
	`

	if _, err := db.Exec(conversationsTableSQL); err != nil {
		return fmt.Errorf("error creating conversations table: %w", err)
	}

	// Add response_format and response_schema columns if they don't exist
	alterTableSQL := `
	ALTER TABLE conversations
	ADD COLUMN IF NOT EXISTS response_format VARCHAR(10) DEFAULT 'text',
	ADD COLUMN IF NOT EXISTS response_schema TEXT;
	`

	if _, err := db.Exec(alterTableSQL); err != nil {
		return fmt.Errorf("error altering conversations table: %w", err)
	}

	// Create messages table
	messagesTableSQL := `
	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY,
		conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		role VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
	`

	if _, err := db.Exec(messagesTableSQL); err != nil {
		return fmt.Errorf("error creating messages table: %w", err)
	}

	// Add model column if it doesn't exist
	alterMessagesTableSQL := `
	ALTER TABLE messages
	ADD COLUMN IF NOT EXISTS model VARCHAR(255);
	`

	if _, err := db.Exec(alterMessagesTableSQL); err != nil {
		return fmt.Errorf("error altering messages table: %w", err)
	}

	// Add temperature column if it doesn't exist
	alterMessagesTemperatureSQL := `
	ALTER TABLE messages
	ADD COLUMN IF NOT EXISTS temperature REAL;
	`

	if _, err := db.Exec(alterMessagesTemperatureSQL); err != nil {
		return fmt.Errorf("error altering messages table for temperature: %w", err)
	}

	// Add usage tracking columns if they don't exist
	alterMessagesUsageSQL := `
	ALTER TABLE messages
	ADD COLUMN IF NOT EXISTS generation_id VARCHAR(255),
	ADD COLUMN IF NOT EXISTS prompt_tokens INTEGER,
	ADD COLUMN IF NOT EXISTS completion_tokens INTEGER,
	ADD COLUMN IF NOT EXISTS total_tokens INTEGER,
	ADD COLUMN IF NOT EXISTS total_cost REAL,
	ADD COLUMN IF NOT EXISTS latency INTEGER,
	ADD COLUMN IF NOT EXISTS generation_time INTEGER;
	`

	if _, err := db.Exec(alterMessagesUsageSQL); err != nil {
		return fmt.Errorf("error altering messages table for usage tracking: %w", err)
	}

	// Add provider column if it doesn't exist
	alterMessagesProviderSQL := `
	ALTER TABLE messages
	ADD COLUMN IF NOT EXISTS provider VARCHAR(50);
	`

	if _, err := db.Exec(alterMessagesProviderSQL); err != nil {
		return fmt.Errorf("error altering messages table for provider: %w", err)
	}

	// Create conversation_summaries table
	summariesTableSQL := `
	CREATE TABLE IF NOT EXISTS conversation_summaries (
		id UUID PRIMARY KEY,
		conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		summary_content TEXT NOT NULL,
		summarized_up_to_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
		usage_count INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_summaries_conversation_id ON conversation_summaries(conversation_id);
	`

	if _, err := db.Exec(summariesTableSQL); err != nil {
		return fmt.Errorf("error creating conversation_summaries table: %w", err)
	}

	// Add active_summary_id column to conversations table if it doesn't exist
	alterConversationsSummarySQL := `
	ALTER TABLE conversations
	ADD COLUMN IF NOT EXISTS active_summary_id UUID REFERENCES conversation_summaries(id) ON DELETE SET NULL;
	`

	if _, err := db.Exec(alterConversationsSummarySQL); err != nil {
		return fmt.Errorf("error altering conversations table for active_summary_id: %w", err)
	}

	return nil
}
