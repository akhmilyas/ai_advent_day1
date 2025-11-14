package app

import (
	"chat-app/internal/config"
	"chat-app/internal/db"
)

// Config holds all application dependencies and configuration
type Config struct {
	// Database interface for data persistence
	DB db.Database
	// Model configuration
	ModelsConfig *config.ModelsConfig
}

// NewConfig creates a new application configuration
func NewConfig(database db.Database, modelsConfig *config.ModelsConfig) *Config {
	return &Config{
		DB:           database,
		ModelsConfig: modelsConfig,
	}
}
