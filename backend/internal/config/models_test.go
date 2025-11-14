package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewModelsConfig_ValidConfig(t *testing.T) {
	// Create a temporary test config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "models.json")

	validJSON := `[
		{
			"id": "meta-llama/llama-3.3-8b-instruct:free",
			"name": "Llama 3.3 8B Instruct (Free)",
			"provider": "Meta",
			"tier": "free"
		},
		{
			"id": "openai/gpt-4",
			"name": "GPT-4",
			"provider": "OpenAI",
			"tier": "paid"
		}
	]`

	err := os.WriteFile(configPath, []byte(validJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := NewModelsConfig(configPath)
	if err != nil {
		t.Errorf("NewModelsConfig() error = %v, want nil", err)
		return
	}

	if config == nil {
		t.Error("NewModelsConfig() returned nil config")
		return
	}

	models := config.GetAvailableModels()
	if len(models) != 2 {
		t.Errorf("GetAvailableModels() returned %d models, want 2", len(models))
	}
}

func TestNewModelsConfig_FileNotFound(t *testing.T) {
	config, err := NewModelsConfig("/nonexistent/path/models.json")
	if err == nil {
		t.Error("NewModelsConfig() error = nil, want error for nonexistent file")
	}

	if config != nil {
		t.Error("NewModelsConfig() returned non-nil config for nonexistent file")
	}
}

func TestNewModelsConfig_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")

	invalidJSON := `{ this is not valid json }`

	err := os.WriteFile(configPath, []byte(invalidJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := NewModelsConfig(configPath)
	if err == nil {
		t.Error("NewModelsConfig() error = nil, want error for invalid JSON")
	}

	if config != nil {
		t.Error("NewModelsConfig() returned non-nil config for invalid JSON")
	}
}

func TestModelsConfig_GetAvailableModels(t *testing.T) {
	config := &ModelsConfig{
		models: []Model{
			{
				ID:       "model-1",
				Name:     "Model 1",
				Provider: "Provider A",
				Tier:     "free",
			},
			{
				ID:       "model-2",
				Name:     "Model 2",
				Provider: "Provider B",
				Tier:     "paid",
			},
		},
	}

	models := config.GetAvailableModels()

	if len(models) != 2 {
		t.Errorf("GetAvailableModels() returned %d models, want 2", len(models))
	}

	if models[0].ID != "model-1" {
		t.Errorf("GetAvailableModels()[0].ID = %s, want model-1", models[0].ID)
	}

	if models[1].ID != "model-2" {
		t.Errorf("GetAvailableModels()[1].ID = %s, want model-2", models[1].ID)
	}
}

func TestModelsConfig_IsValidModel(t *testing.T) {
	config := &ModelsConfig{
		models: []Model{
			{
				ID:       "meta-llama/llama-3.3-8b-instruct:free",
				Name:     "Llama 3.3 8B",
				Provider: "Meta",
				Tier:     "free",
			},
			{
				ID:       "openai/gpt-4",
				Name:     "GPT-4",
				Provider: "OpenAI",
				Tier:     "paid",
			},
		},
	}

	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{
			name:    "valid model - first in list",
			modelID: "meta-llama/llama-3.3-8b-instruct:free",
			want:    true,
		},
		{
			name:    "valid model - second in list",
			modelID: "openai/gpt-4",
			want:    true,
		},
		{
			name:    "invalid model - not in list",
			modelID: "invalid/model",
			want:    false,
		},
		{
			name:    "invalid model - empty string",
			modelID: "",
			want:    false,
		},
		{
			name:    "invalid model - partial match",
			modelID: "meta-llama",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.IsValidModel(tt.modelID)
			if got != tt.want {
				t.Errorf("IsValidModel(%s) = %v, want %v", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestModelsConfig_GetDefaultModel(t *testing.T) {
	tests := []struct {
		name   string
		config *ModelsConfig
		want   string
	}{
		{
			name: "default model from populated list",
			config: &ModelsConfig{
				models: []Model{
					{
						ID:       "first-model",
						Name:     "First Model",
						Provider: "Provider",
						Tier:     "free",
					},
					{
						ID:       "second-model",
						Name:     "Second Model",
						Provider: "Provider",
						Tier:     "paid",
					},
				},
			},
			want: "first-model",
		},
		{
			name: "default model with one model",
			config: &ModelsConfig{
				models: []Model{
					{
						ID:       "only-model",
						Name:     "Only Model",
						Provider: "Provider",
						Tier:     "free",
					},
				},
			},
			want: "only-model",
		},
		{
			name: "fallback model for empty list",
			config: &ModelsConfig{
				models: []Model{},
			},
			want: "meta-llama/llama-3.3-8b-instruct:free",
		},
		{
			name: "fallback model for nil list",
			config: &ModelsConfig{
				models: nil,
			},
			want: "meta-llama/llama-3.3-8b-instruct:free",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetDefaultModel()
			if got != tt.want {
				t.Errorf("GetDefaultModel() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNewModelsConfig_EmptyArray(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty.json")

	emptyJSON := `[]`

	err := os.WriteFile(configPath, []byte(emptyJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := NewModelsConfig(configPath)
	if err != nil {
		t.Errorf("NewModelsConfig() error = %v, want nil for empty array", err)
		return
	}

	if config == nil {
		t.Error("NewModelsConfig() returned nil config for empty array")
		return
	}

	models := config.GetAvailableModels()
	if len(models) != 0 {
		t.Errorf("GetAvailableModels() returned %d models, want 0", len(models))
	}

	// Should return fallback default
	defaultModel := config.GetDefaultModel()
	if defaultModel != "meta-llama/llama-3.3-8b-instruct:free" {
		t.Errorf("GetDefaultModel() = %s, want fallback default", defaultModel)
	}
}

func TestModel_FieldValues(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "models.json")

	testJSON := `[
		{
			"id": "test-id",
			"name": "Test Name",
			"provider": "Test Provider",
			"tier": "test-tier"
		}
	]`

	err := os.WriteFile(configPath, []byte(testJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := NewModelsConfig(configPath)
	if err != nil {
		t.Fatalf("NewModelsConfig() error = %v", err)
	}

	models := config.GetAvailableModels()
	if len(models) != 1 {
		t.Fatalf("GetAvailableModels() returned %d models, want 1", len(models))
	}

	model := models[0]

	if model.ID != "test-id" {
		t.Errorf("Model.ID = %s, want test-id", model.ID)
	}

	if model.Name != "Test Name" {
		t.Errorf("Model.Name = %s, want Test Name", model.Name)
	}

	if model.Provider != "Test Provider" {
		t.Errorf("Model.Provider = %s, want Test Provider", model.Provider)
	}

	if model.Tier != "test-tier" {
		t.Errorf("Model.Tier = %s, want test-tier", model.Tier)
	}
}
