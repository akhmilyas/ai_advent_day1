package config

import (
	"chat-app/internal/logger"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// AppConfig holds all application configuration
type AppConfig struct {
	Server   ServerConfig
	Database DatabaseConfig
	LLM      LLMConfig
	Auth     AuthConfig
	Models   *ModelsConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	OpenRouterAPIKey    string
	DefaultSystemPrompt string
	TextTopP            float64
	TextTopK            int
	StructuredTopP      float64
	StructuredTopK      int
	SummarizationPrompt string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret       []byte
	TokenExpiration time.Duration
}

// LoadConfig loads and validates application configuration from environment
func LoadConfig() (*AppConfig, error) {
	config := &AppConfig{}

	// Load Server config
	config.Server = ServerConfig{
		Port: getEnvOrDefault("SERVER_PORT", "8080"),
	}

	// Load Database config
	config.Database = DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "postgres"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
		Name:     getEnvOrDefault("DB_NAME", "chatapp"),
		SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
	}

	// Load LLM config
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		logger.Log.Warn("OPENROUTER_API_KEY environment variable not set")
	}

	config.LLM = LLMConfig{
		OpenRouterAPIKey:    apiKey,
		DefaultSystemPrompt: getEnvOrDefault("OPENROUTER_SYSTEM_PROMPT", "You are a helpful assistant."),
		TextTopP:            getEnvAsFloat("OPENROUTER_TEXT_TOP_P", 0.9),
		TextTopK:            getEnvAsInt("OPENROUTER_TEXT_TOP_K", 40),
		StructuredTopP:      getEnvAsFloat("OPENROUTER_STRUCTURED_TOP_P", 0.8),
		StructuredTopK:      getEnvAsInt("OPENROUTER_STRUCTURED_TOP_K", 20),
		SummarizationPrompt: getEnvOrDefault("OPENROUTER_SUMMARIZATION_PROMPT", getDefaultSummarizationPrompt()),
	}

	// Load Auth config
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable must be set")
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters (current length: %d)", len(jwtSecret))
	}

	config.Auth = AuthConfig{
		JWTSecret:       []byte(jwtSecret),
		TokenExpiration: getEnvAsDuration("JWT_TOKEN_EXPIRATION", 24*time.Hour),
	}

	// Load Models config
	modelsConfigPath := getEnvOrDefault("MODELS_CONFIG_PATH", filepath.Join("backend", "config", "models.json"))
	modelsConfig, err := NewModelsConfig(modelsConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load models config: %w", err)
	}
	config.Models = modelsConfig

	return config, nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"key": key, "default": defaultValue}).Warn("Invalid integer value, using default")
		return defaultValue
	}
	return value
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"key": key, "default": defaultValue}).Warn("Invalid float value, using default")
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"key": key, "default": defaultValue}).Warn("Invalid duration value, using default")
		return defaultValue
	}
	return value
}

func getDefaultSummarizationPrompt() string {
	return `You are a conversation summarizer. Your task is to create a concise summary of the conversation provided.

Instructions:
1. Capture the main topics discussed
2. Note key decisions or conclusions
3. Preserve important context needed for future messages
4. Keep the summary brief but informative
5. Use clear, neutral language

Provide only the summary, without any preamble or additional commentary.`
}
