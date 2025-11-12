package llm

import (
	"fmt"
	"log"
)

// ProviderType represents the type of LLM provider
type ProviderType string

const (
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderGenkit     ProviderType = "genkit"
)

// ParseProviderType parses a string into a ProviderType
func ParseProviderType(s string) (ProviderType, error) {
	switch s {
	case "openrouter", "":
		return ProviderOpenRouter, nil
	case "genkit":
		return ProviderGenkit, nil
	default:
		return "", fmt.Errorf("unknown provider type: %s", s)
	}
}

// NewLLMProvider creates a new LLM provider based on the specified type
func NewLLMProvider(providerType ProviderType) (LLMProvider, error) {
	switch providerType {
	case ProviderOpenRouter:
		log.Printf("[Factory] Creating OpenRouter provider")
		return NewOpenRouterProvider(), nil
	case ProviderGenkit:
		log.Printf("[Factory] Creating Genkit provider")
		return NewGenkitProvider()
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetProviderFromString creates a provider from a string parameter
// Returns OpenRouter provider by default if string is empty or invalid
func GetProviderFromString(provider string) LLMProvider {
	providerType, err := ParseProviderType(provider)
	if err != nil {
		log.Printf("[Factory] Invalid provider '%s', defaulting to OpenRouter: %v", provider, err)
		providerType = ProviderOpenRouter
	}

	llmProvider, err := NewLLMProvider(providerType)
	if err != nil {
		log.Printf("[Factory] Error creating %s provider, falling back to OpenRouter: %v", providerType, err)
		return NewOpenRouterProvider()
	}

	return llmProvider
}
