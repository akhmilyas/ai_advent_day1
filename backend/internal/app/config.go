package app

import (
	"chat-app/internal/config"
	"chat-app/internal/repository/db"
)

// Config holds all application dependencies and configuration
type Config struct {
	// Database interface for data persistence
	DB db.Database
	// Centralized application configuration
	AppConfig *config.AppConfig
}

// NewConfig creates a new application configuration
func NewConfig(database db.Database, appConfig *config.AppConfig) *Config {
	return &Config{
		DB:        database,
		AppConfig: appConfig,
	}
}

// Helper methods for backward compatibility
func (c *Config) ModelsConfig() *config.ModelsConfig {
	return c.AppConfig.Models
}
