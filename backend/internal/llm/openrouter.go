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
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

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

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
		Delta   Message `json:"delta"`
	} `json:"choices"`
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
		systemPrompt = systemPrompt + "\n\nAdditional instructions: " + customPrompt
	}
	// Prepend system message to the conversation history
	return append([]Message{{Role: "system", Content: systemPrompt}}, messages...)
}

// ChatWithHistory sends a chat request with conversation history and returns the full response
func ChatWithHistory(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
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
			RequireParameters: true,
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

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// ChatWithHistoryStream sends a chat request with conversation history and streams the response
func ChatWithHistoryStream(messages []Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan string, error) {
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
			RequireParameters: true,
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
	chunks := make(chan string)

	// Start reading stream in a goroutine
	go func() {
		defer resp.Body.Close()
		defer close(chunks)

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

				// Extract content from delta field (streaming responses use delta)
				if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
					chunk := streamResp.Choices[0].Delta.Content
					chunks <- chunk
					log.Printf("[LLM] Stream chunk: %q", chunk)
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("[LLM] Scanner error: %v", err)
		}
	}()

	return chunks, nil
}
