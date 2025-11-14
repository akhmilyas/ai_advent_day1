package postgres

import (
	"chat-app/internal/logger"
	"chat-app/internal/repository/db"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the database
// CreateUser creates a new user with hashed password
func (p *PostgresDB) CreateUser(username, email, password string) (*db.User, error) {
	conn := p.conn

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	userID := uuid.New().String()
	var createdAt string

	query := `
	INSERT INTO users (id, username, email, password_hash)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at
	`

	err = conn.QueryRow(query, userID, username, email, string(hashedPassword)).Scan(&userID, &createdAt)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_username_key\"" {
			return nil, fmt.Errorf("username already exists")
		}
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	logger.Log.WithFields(logrus.Fields{"username": username, "user_id": userID}).Info("Created new user")

	return &db.User{
		ID:        userID,
		Username:  username,
		Email:     email,
		CreatedAt: createdAt,
	}, nil
}

// GetUserByUsername retrieves a user by username
func (p *PostgresDB) GetUserByUsername(username string) (*db.User, error) {
	conn := p.conn

	var user db.User
	query := `SELECT id, username, email, password_hash, created_at FROM users WHERE username = $1`

	err := conn.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	return &user, nil
}

// VerifyPassword checks if the provided password matches the user's hashed password
func VerifyPassword(user *db.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// SeedDemoUser creates the demo user if it doesn't exist
func SeedDemoUser(database db.Database) error {
	// Check if demo user already exists
	_, err := database.GetUserByUsername("demo")
	if err == nil {
		logger.Log.Info("Demo user already exists, skipping seed")
		return nil
	}

	// Create demo user
	_, err = database.CreateUser("demo", "demo@example.com", "demo123")
	if err != nil && err.Error() != "username already exists" {
		return fmt.Errorf("error seeding demo user: %w", err)
	}

	logger.Log.Info("Demo user seeded successfully")
	return nil
}
