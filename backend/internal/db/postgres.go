package db

import (
	"chat-app/internal/config"
	"chat-app/internal/logger"
	"database/sql"
	"fmt"
	"os"
	"sync"

	_ "github.com/lib/pq"
)

var (
	instance *sql.DB
	once     sync.Once
)

// PostgresDB implements the Database interface
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB creates a new PostgresDB instance from the existing connection
func NewPostgresDB() *PostgresDB {
	return &PostgresDB{db: instance}
}

// GetDB returns the singleton database connection
func GetDB() *sql.DB {
	return instance
}

// InitDB initializes the database connection and creates tables
// Deprecated: Use InitDBWithConfig instead
func InitDB() error {
	var err error
	once.Do(func() {
		dsn := getDSN()
		logger.Log.WithField("dsn", dsn).Info("Connecting to PostgreSQL")

		instance, err = sql.Open("postgres", dsn)
		if err != nil {
			err = fmt.Errorf("error opening database: %w", err)
			return
		}

		// Test the connection
		if err = instance.Ping(); err != nil {
			err = fmt.Errorf("error connecting to database: %w", err)
			return
		}

		logger.Log.Info("Successfully connected to PostgreSQL")

		// Create tables
		if err = createTables(); err != nil {
			err = fmt.Errorf("error creating tables: %w", err)
			return
		}

		logger.Log.Info("Tables created/verified")
	})

	return err
}

// InitDBWithConfig initializes the database connection using provided config
func InitDBWithConfig(dbConfig config.DatabaseConfig) error {
	var err error
	once.Do(func() {
		dsn := dbConfig.GetDSN()
		logger.Log.WithField("dsn", dsn).Info("Connecting to PostgreSQL")

		instance, err = sql.Open("postgres", dsn)
		if err != nil {
			err = fmt.Errorf("error opening database: %w", err)
			return
		}

		// Test the connection
		if err = instance.Ping(); err != nil {
			err = fmt.Errorf("error connecting to database: %w", err)
			return
		}

		logger.Log.Info("Successfully connected to PostgreSQL")

		// Create tables
		if err = createTables(); err != nil {
			err = fmt.Errorf("error creating tables: %w", err)
			return
		}

		logger.Log.Info("Tables created/verified")
	})

	return err
}

// CloseDB closes the database connection
func CloseDB() error {
	if instance != nil {
		return instance.Close()
	}
	return nil
}

// getDSN constructs the PostgreSQL connection string
func getDSN() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	database := os.Getenv("DB_NAME")
	if database == "" {
		database = "chatapp"
	}

	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, database, sslMode)
}

// createTables creates the necessary database tables
func createTables() error {
	db := GetDB()

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
