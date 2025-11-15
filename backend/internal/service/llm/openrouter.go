package llm

import (
	"bufio"
	"bytes"
	"chat-app/internal/config"
	"chat-app/internal/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
const openRouterGenerationURL = "https://openrouter.ai/api/v1/generation"

// OpenRouterProvider implements LLMProvider using direct OpenRouter API calls
type OpenRouterProvider struct {
	config *config.LLMConfig
	models *config.ModelsConfig
}

// NewOpenRouterProvider creates a new OpenRouter provider with config
func NewOpenRouterProvider(llmConfig *config.LLMConfig, modelsConfig *config.ModelsConfig) *OpenRouterProvider {
	return &OpenRouterProvider{
		config: llmConfig,
		models: modelsConfig,
	}
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
	GenerationID   string
	Usage          *ResponseUsage
	TotalCost      *float64
	Latency        *int
	GenerationTime *int
}

type StreamChunk struct {
	Content  string
	Metadata *StreamMetadata
	IsDone   bool
}

// Provider helper methods

func (p *OpenRouterProvider) getAPIKey() string {
	return p.config.OpenRouterAPIKey
}

func (p *OpenRouterProvider) getModel() string {
	return p.models.GetDefaultModel()
}

func (p *OpenRouterProvider) getSystemPrompt() string {
	return p.config.DefaultSystemPrompt
}

func (p *OpenRouterProvider) getTopP(format string) *float64 {
	if format == "json" || format == "xml" {
		return &p.config.StructuredTopP
	}
	return &p.config.TextTopP
}

func (p *OpenRouterProvider) getTopK(format string) *int {
	if format == "json" || format == "xml" {
		return &p.config.StructuredTopK
	}
	return &p.config.TextTopK
}

func (p *OpenRouterProvider) getSummarizationPrompt() string {
	return p.config.SummarizationPrompt
}

func (p *OpenRouterProvider) buildMessagesWithHistory(messages []Message, customPrompt string) []Message {
	systemPrompt := p.getSystemPrompt()

	// If custom prompt is provided, append it to the default system prompt
	if customPrompt != "" {
		systemPrompt = systemPrompt + "\n\n" + customPrompt
	}

	// Log the final system prompt
	logger.Log.WithField("prompt_length", len(systemPrompt)).Debug("Using system prompt")

	// Prepend system message to the conversation history
	return append([]Message{{Role: "system", Content: systemPrompt}}, messages...)
}

// buildMessagesWithCustomSystemPrompt builds messages with ONLY the custom prompt (no default)
// Used for summarization where we don't want the default system prompt
func buildMessagesWithCustomSystemPrompt(messages []Message, customPrompt string) []Message {
	// Log the system prompt
	logger.Log.WithField("prompt_length", len(customPrompt)).Debug("Using custom-only system prompt")

	// Prepend system message to the conversation history
	return append([]Message{{Role: "system", Content: customPrompt}}, messages...)
}

// ChatWithHistory sends a chat request with conversation history and returns the full response
func (p *OpenRouterProvider) ChatWithHistory(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
	apiKey := p.getAPIKey()
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	model := modelOverride
	if model == "" {
		model = p.getModel()
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
	}).Info("Calling OpenRouter API")

	messagesWithHistory := p.buildMessagesWithHistory(messages, customSystemPrompt)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messagesWithHistory,
		Stream:      false,
		Temperature: temperature,
		TopP:        p.getTopP(format),
		TopK:        p.getTopK(format),
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

	logger.Log.WithField("response_length", len(body)).Debug("Received raw response")

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := chatResp.Choices[0].Message.Content
	logger.Log.WithField("content_length", len(content)).Debug("Extracted content from response")
	return content, nil
}

// ChatForSummarization sends a chat request for summarization with ONLY the custom prompt (no default system prompt)
func (p *OpenRouterProvider) ChatForSummarization(messages []Message, summarizationPrompt string, modelOverride string, temperature *float64) (string, error) {
	apiKey := p.getAPIKey()
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	model := modelOverride
	if model == "" {
		model = p.getModel()
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	logger.Log.WithFields(logrus.Fields{
		"model": model,
		"temperature": tempStr,
		"message_count": len(messages),
	}).Info("Calling OpenRouter API for summarization")

	messagesWithHistory := buildMessagesWithCustomSystemPrompt(messages, summarizationPrompt)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messagesWithHistory,
		Stream:      false,
		Temperature: temperature,
		TopP:        p.getTopP("text"),
		TopK:        p.getTopK("text"),
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

	logger.Log.WithField("response_length", len(body)).Debug("Received raw summarization response")

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := chatResp.Choices[0].Message.Content
	logger.Log.WithField("content_length", len(content)).Debug("Extracted summarization content")
	return content, nil
}

// ChatWithHistoryStream sends a chat request with conversation history and streams the response
func (p *OpenRouterProvider) ChatWithHistoryStream(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan StreamChunk, error) {
	apiKey := p.getAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	model := modelOverride
	if model == "" {
		model = p.getModel()
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
	}).Info("Calling OpenRouter API (streaming)")

	messagesWithHistory := p.buildMessagesWithHistory(messages, customSystemPrompt)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messagesWithHistory,
		Stream:      true,
		Temperature: temperature,
		TopP:        p.getTopP(format),
		TopK:        p.getTopK(format),
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
					logger.Log.WithError(err).Warn("Error parsing stream chunk")
					continue
				}

				// Capture generation ID if present
				if streamResp.ID != "" && generationID == "" {
					generationID = streamResp.ID
					logger.Log.WithField("generation_id", generationID).Debug("Captured generation ID")
				}

				// Capture usage data if present (sent at end with empty choices)
				if streamResp.Usage != nil {
					usage = streamResp.Usage
					logger.Log.WithFields(logrus.Fields{
					"prompt_tokens": usage.PromptTokens,
					"completion_tokens": usage.CompletionTokens,
					"total_tokens": usage.TotalTokens,
				}).Debug("Captured usage data")
				}

				// Extract content from delta field (streaming responses use delta)
				if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
					chunk := streamResp.Choices[0].Delta.Content
					chunks <- StreamChunk{Content: chunk}
					logger.Log.WithField("chunk_length", len(chunk)).Debug("Stream chunk received")
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
			logger.Log.Debug("Sent final metadata chunk")
		}

		if err := scanner.Err(); err != nil {
			logger.Log.WithError(err).Error("Scanner error during streaming")
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

	apiKey := p.getAPIKey()
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
			logger.Log.WithFields(logrus.Fields{"delay": delay, "attempt": attempt+1, "max_retries": maxRetries}).Info("Retrying cost fetch")
			time.Sleep(delay)
		}

		logger.Log.WithFields(logrus.Fields{"url": url, "attempt": attempt+1, "max_retries": maxRetries}).Debug("Fetching generation cost")

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
		logger.Log.WithFields(logrus.Fields{"status_code": resp.StatusCode, "response_length": len(body)}).Debug("Raw generation API response")

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

		logger.Log.WithFields(logrus.Fields{
			"cost":              genResp.Data.TotalCost,
			"prompt_tokens":     genResp.Data.NativeTokensPrompt,
			"completion_tokens": genResp.Data.NativeTokensCompletion,
			"latency_ms":        genResp.Data.Latency,
			"generation_time_ms": genResp.Data.GenerationTime,
		}).Info("Fetched generation cost data")

		return &genResp.Data, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// GetDefaultModel returns the default model for OpenRouter provider
func (p *OpenRouterProvider) GetDefaultModel() string {
	return p.getModel()
}
