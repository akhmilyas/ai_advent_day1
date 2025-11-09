package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Model represents an available LLM model
type Model struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Tier     string `json:"tier"`
}

var availableModels []Model

// LoadModels loads the available models from the config file
func LoadModels(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &availableModels)
	if err != nil {
		return err
	}

	return nil
}

// GetAvailableModels returns the list of available models
func GetAvailableModels() []Model {
	return availableModels
}

// IsValidModel checks if a model ID is in the list of available models
func IsValidModel(modelID string) bool {
	for _, model := range availableModels {
		if model.ID == modelID {
			return true
		}
	}
	return false
}

// GetDefaultModelPath returns the default path to the models config file
func GetDefaultModelPath() string {
	return filepath.Join("backend", "config", "models.json")
}
