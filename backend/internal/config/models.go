package config

import (
	"encoding/json"
	"os"
)

// Model represents an available LLM model
type Model struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Tier     string `json:"tier"`
}

// ModelsConfig holds the available models configuration
type ModelsConfig struct {
	models []Model
}

// NewModelsConfig creates a new models configuration from a file
func NewModelsConfig(configPath string) (*ModelsConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var models []Model
	err = json.Unmarshal(data, &models)
	if err != nil {
		return nil, err
	}

	return &ModelsConfig{models: models}, nil
}

// GetAvailableModels returns the list of available models
func (mc *ModelsConfig) GetAvailableModels() []Model {
	return mc.models
}

// IsValidModel checks if a model ID is in the list of available models
func (mc *ModelsConfig) IsValidModel(modelID string) bool {
	for _, model := range mc.models {
		if model.ID == modelID {
			return true
		}
	}
	return false
}

// GetDefaultModel returns the first model as the default
func (mc *ModelsConfig) GetDefaultModel() string {
	if len(mc.models) > 0 {
		return mc.models[0].ID
	}
	// Fallback in case no models are configured (shouldn't happen)
	return "meta-llama/llama-3.3-8b-instruct:free"
}
