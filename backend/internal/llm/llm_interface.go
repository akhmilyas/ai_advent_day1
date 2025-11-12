package llm

// LLMProvider defines the interface for LLM providers (OpenRouter direct API, Genkit, etc.)
type LLMProvider interface {
	// ChatWithHistory sends a chat request with conversation history and returns the full response
	ChatWithHistory(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error)

	// ChatWithHistoryStream sends a chat request with conversation history and streams the response
	ChatWithHistoryStream(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan StreamChunk, error)

	// FetchGenerationCost fetches cost information for a generation (if supported)
	FetchGenerationCost(generationID string) (*GenerationData, error)

	// GetDefaultModel returns the default model for this provider
	GetDefaultModel() string
}
