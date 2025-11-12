package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go"
)

// GenkitProvider implements LLMProvider using Firebase Genkit with OpenRouter via compat_oai
type GenkitProvider struct {
	genkit *genkit.Genkit
	mu     sync.Mutex
}

// NewGenkitProvider creates a new Genkit provider instance configured for OpenRouter
func NewGenkitProvider() (*GenkitProvider, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	ctx := context.Background()

	// Get default model from config
	defaultModel := GetModel()

	// Initialize Genkit with OpenRouter plugin
	g := genkit.Init(ctx,
		genkit.WithPlugins(&compat_oai.OpenAICompatible{
			Provider: "openrouter",
			APIKey:   apiKey,
			BaseURL:  "https://openrouter.ai/api/v1",
		}),
		genkit.WithDefaultModel("openrouter/"+defaultModel),
	)

	log.Printf("[Genkit] Initialized with OpenRouter provider, default model: %s", defaultModel)

	return &GenkitProvider{
		genkit: g,
	}, nil
}

// ChatWithHistory sends a chat request with conversation history and returns the full response
func (p *GenkitProvider) ChatWithHistory(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
	model := modelOverride
	if model == "" {
		model = GetModel()
	}

	// Ensure model has openrouter/ prefix
	if !strings.HasPrefix(model, "openrouter/") {
		model = "openrouter/" + model
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	log.Printf("[Genkit] Calling with model: %s, format: %s, temperature: %s, message history count: %d", model, format, tempStr, len(messages))

	messagesWithHistory := buildMessagesWithHistory(messages, customSystemPrompt)

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
	topP := GetTopP(format)
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
		model = GetModel()
	}

	// Ensure model has openrouter/ prefix
	if !strings.HasPrefix(model, "openrouter/") {
		model = "openrouter/" + model
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	log.Printf("[Genkit] Calling (streaming) with model: %s, format: %s, temperature: %s, message history count: %d", model, format, tempStr, len(messages))

	messagesWithHistory := buildMessagesWithHistory(messages, customSystemPrompt)

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
	topP := GetTopP(format)
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
						log.Printf("[Genkit] Stream chunk: %q", text)
					}
				}
				return nil
			}),
		)

		if err != nil {
			log.Printf("[Genkit] Stream error: %v", err)
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
			log.Printf("[Genkit] Usage - Prompt: %d, Completion: %d, Total: %d",
				usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
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

		log.Printf("[Genkit] Stream completed, full response length: %d", len(fullResponse.String()))
	}()

	return chunks, nil
}

// FetchGenerationCost fetches cost information for a generation
// Note: Genkit doesn't provide generation cost tracking in the same way as OpenRouter
func (p *GenkitProvider) FetchGenerationCost(generationID string) (*GenerationData, error) {
	// Genkit via compat_oai doesn't expose OpenRouter's generation endpoint
	// We could potentially track this via OpenTelemetry traces if needed
	log.Printf("[Genkit] FetchGenerationCost not supported for Genkit provider")
	return nil, fmt.Errorf("generation cost tracking not supported for Genkit provider")
}

// GetDefaultModel returns the default model for Genkit provider
func (p *GenkitProvider) GetDefaultModel() string {
	return GetModel()
}

// FetchGenerationCostViaOpenRouter attempts to fetch generation cost using direct OpenRouter API
// This is a helper function that can be called after Genkit streaming if the generation ID is available
func (p *GenkitProvider) FetchGenerationCostViaOpenRouter(generationID string) (*GenerationData, error) {
	if generationID == "" {
		return nil, fmt.Errorf("generation ID is empty")
	}

	apiKey := GetAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	url := fmt.Sprintf("%s?id=%s", openRouterGenerationURL, generationID)
	log.Printf("[Genkit] Fetching generation cost from OpenRouter API: %s", url)

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
