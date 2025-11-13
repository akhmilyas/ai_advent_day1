package llm

import (
	"bufio"
	"bytes"
	"chat-app/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
const openRouterGenerationURL = "https://openrouter.ai/api/v1/generation"

// OpenRouterProvider implements LLMProvider using direct OpenRouter API calls
type OpenRouterProvider struct{}

// NewOpenRouterProvider creates a new OpenRouter provider instance
func NewOpenRouterProvider() *OpenRouterProvider {
	return &OpenRouterProvider{}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Provider struct {
	RequireParameters bool `json:"require_parameters,omitempty"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature *float64  `json:"temperature,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	TopK        *int      `json:"top_k,omitempty"`
	Provider    *Provider `json:"provider,omitempty"`
}

type ResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message Message `json:"message"`
		Delta   Message `json:"delta"`
	} `json:"choices"`
	Usage *ResponseUsage `json:"usage,omitempty"`
}

type StreamMetadata struct {
	GenerationID string
	Usage        *ResponseUsage
}

type StreamChunk struct {
	Content  string
	Metadata *StreamMetadata
	IsDone   bool
}

func GetAPIKey() string {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: OPENROUTER_API_KEY environment variable not set")
	}
	return apiKey
}

func GetModel() string {
	// Get the first model from config as default
	models := config.GetAvailableModels()
	if len(models) > 0 {
		return models[0].ID
	}
	// Fallback in case no models are configured (shouldn't happen)
	log.Println("[LLM] Warning: No models configured, using fallback")
	return "meta-llama/llama-3.3-8b-instruct:free"
}

func GetSystemPrompt() string {
	systemPrompt := os.Getenv("OPENROUTER_SYSTEM_PROMPT")
	if systemPrompt == "" {
		// Default system prompt
		systemPrompt = "You are a helpful assistant."
	}
	return systemPrompt
}

func GetTopP(format string) *float64 {
	var envVar string

	// Determine which environment variable to use based on format
	if format == "json" || format == "xml" {
		envVar = "OPENROUTER_STRUCTURED_TOP_P"
	} else {
		envVar = "OPENROUTER_TEXT_TOP_P"
	}

	// Get value from environment
	topPStr := os.Getenv(envVar)
	if topPStr != "" {
		var topP float64
		if _, err := fmt.Sscanf(topPStr, "%f", &topP); err == nil {
			return &topP
		}
	}

	// Should not reach here if .env is configured, but return nil as fallback
	return nil
}

func GetTopK(format string) *int {
	var envVar string

	// Determine which environment variable to use based on format
	if format == "json" || format == "xml" {
		envVar = "OPENROUTER_STRUCTURED_TOP_K"
	} else {
		envVar = "OPENROUTER_TEXT_TOP_K"
	}

	// Get value from environment
	topKStr := os.Getenv(envVar)
	if topKStr != "" {
		var topK int
		if _, err := fmt.Sscanf(topKStr, "%d", &topK); err == nil {
			return &topK
		}
	}

	// Should not reach here if .env is configured, but return nil as fallback
	return nil
}

func buildMessagesWithHistory(messages []Message, customPrompt string) []Message {
	systemPrompt := GetSystemPrompt()

	// If custom prompt is provided, append it to the default system prompt
	if customPrompt != "" {
		systemPrompt = systemPrompt + "\n\n" + customPrompt
	}

	// Log the final system prompt
	log.Printf("[LLM] System prompt (length: %d): %s", len(systemPrompt), systemPrompt)

	// Prepend system message to the conversation history
	return append([]Message{{Role: "system", Content: systemPrompt}}, messages...)
}

// buildMessagesWithCustomSystemPrompt builds messages with ONLY the custom prompt (no default)
// Used for summarization where we don't want the default system prompt
func buildMessagesWithCustomSystemPrompt(messages []Message, customPrompt string) []Message {
	// Log the system prompt
	log.Printf("[LLM] Using custom-only system prompt (length: %d): %s", len(customPrompt), customPrompt)

	// Prepend system message to the conversation history
	return append([]Message{{Role: "system", Content: customPrompt}}, messages...)
}

// ChatWithHistory sends a chat request with conversation history and returns the full response
func (p *OpenRouterProvider) ChatWithHistory(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	model := modelOverride
	if model == "" {
		model = GetModel()
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	log.Printf("[LLM] Calling OpenRouter API with model: %s, format: %s, temperature: %s, message history count: %d", model, format, tempStr, len(messages))

	messagesWithHistory := buildMessagesWithHistory(messages, customSystemPrompt)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messagesWithHistory,
		Stream:      false,
		Temperature: temperature,
		TopP:        GetTopP(format),
		TopK:        GetTopK(format),
		Provider: &Provider{
			RequireParameters: false,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", openRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "http://localhost:3000")
	req.Header.Set("X-Title", "Chat App")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	log.Printf("[LLM] Raw response body: %s", string(body))

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := chatResp.Choices[0].Message.Content
	log.Printf("[LLM] Extracted content length: %d", len(content))
	return content, nil
}

// ChatForSummarization sends a chat request for summarization with ONLY the custom prompt (no default system prompt)
func (p *OpenRouterProvider) ChatForSummarization(messages []Message, summarizationPrompt string, modelOverride string, temperature *float64) (string, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	model := modelOverride
	if model == "" {
		model = GetModel()
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	log.Printf("[LLM] Calling OpenRouter API for summarization with model: %s, temperature: %s, message history count: %d", model, tempStr, len(messages))

	messagesWithHistory := buildMessagesWithCustomSystemPrompt(messages, summarizationPrompt)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messagesWithHistory,
		Stream:      false,
		Temperature: temperature,
		TopP:        GetTopP("text"),
		TopK:        GetTopK("text"),
		Provider: &Provider{
			RequireParameters: false,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", openRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "http://localhost:3000")
	req.Header.Set("X-Title", "Chat App")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	log.Printf("[LLM] Raw summarization response body: %s", string(body))

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := chatResp.Choices[0].Message.Content
	log.Printf("[LLM] Extracted summarization content length: %d", len(content))
	return content, nil
}

// ChatWithHistoryStream sends a chat request with conversation history and streams the response
func (p *OpenRouterProvider) ChatWithHistoryStream(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan StreamChunk, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	model := modelOverride
	if model == "" {
		model = GetModel()
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	log.Printf("[LLM] Calling OpenRouter API (streaming) with model: %s, format: %s, temperature: %s, message history count: %d", model, format, tempStr, len(messages))

	messagesWithHistory := buildMessagesWithHistory(messages, customSystemPrompt)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messagesWithHistory,
		Stream:      true,
		Temperature: temperature,
		TopP:        GetTopP(format),
		TopK:        GetTopK(format),
		Provider: &Provider{
			RequireParameters: false,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", openRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "http://localhost:3000")
	req.Header.Set("X-Title", "Chat App")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Create channel to stream chunks
	chunks := make(chan StreamChunk)

	// Start reading stream in a goroutine
	go func() {
		defer resp.Body.Close()
		defer close(chunks)

		var generationID string
		var usage *ResponseUsage

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines and [DONE] markers
			if line == "" || line == "data: [DONE]" {
				continue
			}

			// Parse SSE event format: "data: {json}"
			if strings.HasPrefix(line, "data: ") {
				jsonStr := strings.TrimPrefix(line, "data: ")

				var streamResp ChatResponse
				if err := json.Unmarshal([]byte(jsonStr), &streamResp); err != nil {
					log.Printf("[LLM] Error parsing stream chunk: %v", err)
					continue
				}

				// Capture generation ID if present
				if streamResp.ID != "" && generationID == "" {
					generationID = streamResp.ID
					log.Printf("[LLM] Captured generation ID: %s", generationID)
				}

				// Capture usage data if present (sent at end with empty choices)
				if streamResp.Usage != nil {
					usage = streamResp.Usage
					log.Printf("[LLM] Captured usage: prompt=%d, completion=%d, total=%d",
						usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
				}

				// Extract content from delta field (streaming responses use delta)
				if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
					chunk := streamResp.Choices[0].Delta.Content
					chunks <- StreamChunk{Content: chunk}
					log.Printf("[LLM] Stream chunk: %q", chunk)
				}
			}
		}

		// Send final metadata chunk
		if generationID != "" || usage != nil {
			chunks <- StreamChunk{
				Metadata: &StreamMetadata{
					GenerationID: generationID,
					Usage:        usage,
				},
				IsDone: true,
			}
			log.Printf("[LLM] Sent final metadata chunk")
		}

		if err := scanner.Err(); err != nil {
			log.Printf("[LLM] Scanner error: %v", err)
		}
	}()

	return chunks, nil
}

// GenerationData represents cost and usage information from OpenRouter
type GenerationData struct {
	ID                     string  `json:"id"`
	TotalCost              float64 `json:"total_cost"`
	TokensPrompt           int     `json:"tokens_prompt"`
	TokensCompletion       int     `json:"tokens_completion"`
	NativeTokensPrompt     int     `json:"native_tokens_prompt"`
	NativeTokensCompletion int     `json:"native_tokens_completion"`
	Latency                int     `json:"latency"`        // Time to first token in milliseconds
	GenerationTime         int     `json:"generation_time"` // Total generation time in milliseconds
}

type GenerationResponse struct {
	Data GenerationData `json:"data"`
}

// FetchGenerationCost fetches cost information for a generation from OpenRouter
// with retry logic to handle timing delays in data availability
func (p *OpenRouterProvider) FetchGenerationCost(generationID string) (*GenerationData, error) {
	if generationID == "" {
		return nil, fmt.Errorf("generation ID is empty")
	}

	apiKey := GetAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	url := fmt.Sprintf("%s?id=%s", openRouterGenerationURL, generationID)

	// Retry configuration: 3 attempts with exponential backoff (500ms, 1s, 2s)
	maxRetries := 3
	baseDelay := 500 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1)) // Exponential: 500ms, 1s, 2s
			log.Printf("[LLM] Retrying cost fetch in %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}

		log.Printf("[LLM] Fetching generation cost from: %s (attempt %d/%d)", url, attempt+1, maxRetries)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error sending request: %w", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Log raw response for debugging
		log.Printf("[LLM] Raw generation API response (status %d): %s", resp.StatusCode, string(body))

		if resp.StatusCode == http.StatusNotFound {
			// 404 means data not ready yet, retry
			lastErr = fmt.Errorf("generation not found yet (status 404)")
			continue
		} else if resp.StatusCode != http.StatusOK {
			// Other errors are not retryable
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var genResp GenerationResponse
		if err := json.Unmarshal(body, &genResp); err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}

		log.Printf("[LLM] Generation - cost: $%.6f, native_tokens: prompt=%d, completion=%d, latency: %dms, generation_time: %dms",
			genResp.Data.TotalCost, genResp.Data.NativeTokensPrompt, genResp.Data.NativeTokensCompletion,
			genResp.Data.Latency, genResp.Data.GenerationTime)

		return &genResp.Data, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// GetDefaultModel returns the default model for OpenRouter provider
func (p *OpenRouterProvider) GetDefaultModel() string {
	return GetModel()
}
