package llm

import (
	"bytes"
	"chat-app/internal/config"
	"chat-app/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go"
	"github.com/sirupsen/logrus"
)

// GenkitProvider implements LLMProvider using Firebase Genkit with OpenRouter via compat_oai
type GenkitProvider struct {
	genkit *genkit.Genkit
	config *config.LLMConfig
	models *config.ModelsConfig
	mu     sync.Mutex
}

// NewGenkitProvider creates a new Genkit provider instance configured for OpenRouter
func NewGenkitProvider(llmConfig *config.LLMConfig, modelsConfig *config.ModelsConfig) (*GenkitProvider, error) {
	apiKey := llmConfig.OpenRouterAPIKey
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	ctx := context.Background()

	// Get default model from config
	defaultModel := modelsConfig.GetDefaultModel()

	// Initialize Genkit with OpenRouter plugin
	g := genkit.Init(ctx,
		genkit.WithPlugins(&compat_oai.OpenAICompatible{
			Provider: "openrouter",
			APIKey:   apiKey,
			BaseURL:  "https://openrouter.ai/api/v1",
		}),
		genkit.WithDefaultModel("openrouter/"+defaultModel),
	)

	logger.Log.WithField("default_model", defaultModel).Info("Initialized Genkit with OpenRouter provider")

	return &GenkitProvider{
		genkit: g,
		config: llmConfig,
		models: modelsConfig,
		mu:     sync.Mutex{},
	}, nil
}

// Helper methods
func (p *GenkitProvider) getModel() string {
	return p.models.GetDefaultModel()
}

func (p *GenkitProvider) getAPIKey() string {
	return p.config.OpenRouterAPIKey
}

func (p *GenkitProvider) getSystemPrompt() string {
	return p.config.DefaultSystemPrompt
}

func (p *GenkitProvider) getTopP(format string) *float64 {
	if format == "json" || format == "xml" {
		return &p.config.StructuredTopP
	}
	return &p.config.TextTopP
}

func (p *GenkitProvider) getTopK(format string) *int {
	if format == "json" || format == "xml" {
		return &p.config.StructuredTopK
	}
	return &p.config.TextTopK
}

func (p *GenkitProvider) buildMessagesWithHistory(messages []Message, customPrompt string) []Message {
	systemPrompt := p.getSystemPrompt()

	if customPrompt != "" {
		systemPrompt = systemPrompt + "\n\n" + customPrompt
	}

	logger.Log.WithField("prompt_length", len(systemPrompt)).Debug("Using system prompt")
	return append([]Message{{Role: "system", Content: systemPrompt}}, messages...)
}

// ChatWithHistory sends a chat request with conversation history and returns the full response
func (p *GenkitProvider) ChatWithHistory(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
	model := modelOverride
	if model == "" {
		model = p.getModel()
	}

	// Ensure model has openrouter/ prefix
	if !strings.HasPrefix(model, "openrouter/") {
		model = "openrouter/" + model
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	logger.Log.WithFields(logrus.Fields{
		"model": model,
		"format": format,
		"temperature": tempStr,
		"message_count": len(messages),
	}).Info("Calling Genkit")

	messagesWithHistory := p.buildMessagesWithHistory(messages, customSystemPrompt)

	// Convert messages to Genkit format
	var genkitMessages []*ai.Message
	for _, msg := range messagesWithHistory {
		role := ai.Role(msg.Role)
		genkitMessages = append(genkitMessages, &ai.Message{
			Role:    role,
			Content: []*ai.Part{ai.NewTextPart(msg.Content)},
		})
	}

	// Build config using OpenAI ChatCompletionNewParams
	config := &openai.ChatCompletionNewParams{}

	// Set temperature
	if temperature != nil {
		config.Temperature = openai.Float(*temperature)
	}

	// Set top_p based on format
	topP := p.getTopP(format)
	if topP != nil {
		config.TopP = openai.Float(*topP)
	}

	// Note: OpenAI API doesn't support top_k, so we skip it for Genkit

	// Generate response
	ctx := context.Background()
	resp, err := genkit.Generate(ctx, p.genkit,
		ai.WithMessages(genkitMessages...),
		ai.WithModelName(model),
		ai.WithConfig(config),
	)

	if err != nil {
		return "", fmt.Errorf("genkit generation failed: %w", err)
	}

	return resp.Text(), nil
}

// ChatWithHistoryStream sends a chat request with conversation history and streams the response
func (p *GenkitProvider) ChatWithHistoryStream(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan StreamChunk, error) {
	model := modelOverride
	if model == "" {
		model = p.getModel()
	}

	// Ensure model has openrouter/ prefix
	if !strings.HasPrefix(model, "openrouter/") {
		model = "openrouter/" + model
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	logger.Log.WithFields(logrus.Fields{
		"model": model,
		"format": format,
		"temperature": tempStr,
		"message_count": len(messages),
	}).Info("Calling Genkit (streaming)")

	messagesWithHistory := p.buildMessagesWithHistory(messages, customSystemPrompt)

	// Convert messages to Genkit format
	var genkitMessages []*ai.Message
	for _, msg := range messagesWithHistory {
		role := ai.Role(msg.Role)
		genkitMessages = append(genkitMessages, &ai.Message{
			Role:    role,
			Content: []*ai.Part{ai.NewTextPart(msg.Content)},
		})
	}

	// Build config using OpenAI ChatCompletionNewParams
	config := &openai.ChatCompletionNewParams{}

	// Set temperature
	if temperature != nil {
		config.Temperature = openai.Float(*temperature)
	}

	// Set top_p based on format
	topP := p.getTopP(format)
	if topP != nil {
		config.TopP = openai.Float(*topP)
	}

	// Note: OpenAI API doesn't support top_k, so we skip it for Genkit

	// Create channel to stream chunks
	chunks := make(chan StreamChunk)

	// Start streaming in a goroutine
	go func() {
		defer close(chunks)

		ctx := context.Background()
		var fullResponse strings.Builder

		// Generate with streaming
		resp, err := genkit.Generate(ctx, p.genkit,
			ai.WithMessages(genkitMessages...),
			ai.WithModelName(model),
			ai.WithConfig(config),
			ai.WithStreaming(func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
				// Extract text from chunk
				for _, part := range chunk.Content {
					if part.IsText() {
						text := part.Text
						fullResponse.WriteString(text)
						chunks <- StreamChunk{Content: text}
						logger.Log.WithField("chunk_length", len(text)).Debug("Stream chunk received")
					}
				}
				return nil
			}),
		)

		if err != nil {
			logger.Log.WithError(err).Error("Stream error")
			return
		}

		// Extract usage information from Genkit response
		var usage *ResponseUsage
		if resp.Usage != nil {
			usage = &ResponseUsage{
				PromptTokens:     int(resp.Usage.InputTokens),
				CompletionTokens: int(resp.Usage.OutputTokens),
				TotalTokens:      int(resp.Usage.TotalTokens),
			}
			logger.Log.WithFields(logrus.Fields{
				"prompt_tokens":     usage.PromptTokens,
				"completion_tokens": usage.CompletionTokens,
				"total_tokens":      usage.TotalTokens,
			}).Debug("Captured usage data")
		}

		// Send final metadata chunk with usage data
		// Note: Genkit doesn't expose OpenRouter's generation ID for cost tracking
		chunks <- StreamChunk{
			Metadata: &StreamMetadata{
				GenerationID: "", // Not available from Genkit/compat_oai
				Usage:        usage,
			},
			IsDone: true,
		}

		logger.Log.WithField("response_length", len(fullResponse.String())).Debug("Stream completed")
	}()

	return chunks, nil
}

// FetchGenerationCost fetches cost information for a generation
// Note: Genkit doesn't provide generation cost tracking in the same way as OpenRouter
func (p *GenkitProvider) FetchGenerationCost(generationID string) (*GenerationData, error) {
	// Genkit via compat_oai doesn't expose OpenRouter's generation endpoint
	// We could potentially track this via OpenTelemetry traces if needed
	logger.Log.Warn("FetchGenerationCost not supported for Genkit provider")
	return nil, fmt.Errorf("generation cost tracking not supported for Genkit provider")
}

// GetDefaultModel returns the default model for Genkit provider
func (p *GenkitProvider) GetDefaultModel() string {
	return p.getModel()
}

// FetchGenerationCostViaOpenRouter attempts to fetch generation cost using direct OpenRouter API
// This is a helper function that can be called after Genkit streaming if the generation ID is available
func (p *GenkitProvider) FetchGenerationCostViaOpenRouter(generationID string) (*GenerationData, error) {
	if generationID == "" {
		return nil, fmt.Errorf("generation ID is empty")
	}

	apiKey := p.getAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	url := fmt.Sprintf("%s?id=%s", openRouterGenerationURL, generationID)
	logger.Log.WithField("url", url).Debug("Fetching generation cost from OpenRouter API")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var genResp GenerationResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &genResp.Data, nil
}

// ParseGenerationIDFromStream attempts to extract generation ID from response headers or body
// This is needed because compat_oai may not expose the x-request-id header directly
func ParseGenerationIDFromResponse(body []byte) string {
	// Try to parse as JSON and look for an ID field
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		if id, ok := result["id"].(string); ok {
			return id
		}
	}
	return ""
}

// ExtractGenerationIDFromSSE extracts the generation ID from SSE stream data
func ExtractGenerationIDFromSSE(line string) string {
	if !strings.HasPrefix(line, "data: ") {
		return ""
	}

	jsonStr := strings.TrimPrefix(line, "data: ")
	if jsonStr == "[DONE]" {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	if id, ok := data["id"].(string); ok {
		return id
	}

	return ""
}

// UnmarshalResponseForID unmarshals a streaming response to extract the ID
func UnmarshalResponseForID(data []byte) string {
	var buf bytes.Buffer
	buf.Write(data)

	var response struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return ""
	}

	return response.ID
}
