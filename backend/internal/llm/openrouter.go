package llm

import (
	"bufio"
	"bytes"
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

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature *float64  `json:"temperature,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	TopK        *int      `json:"top_k,omitempty"`
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
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		// Default to free LLaMA model
		model = "meta-llama/llama-3.3-8b-instruct:free"
	}
	return model
}

func GetSystemPrompt() string {
	systemPrompt := os.Getenv("OPENROUTER_SYSTEM_PROMPT")
	if systemPrompt == "" {
		// Default system prompt
		systemPrompt = "You are a helpful assistant."
	}
	return systemPrompt
}

func GetTemperature() *float64 {
	tempStr := os.Getenv("OPENROUTER_TEMPERATURE")
	if tempStr != "" {
		var temp float64
		if _, err := fmt.Sscanf(tempStr, "%f", &temp); err == nil {
			return &temp
		}
	}
	return nil
}

func GetTopP() *float64 {
	topPStr := os.Getenv("OPENROUTER_TOP_P")
	if topPStr != "" {
		var topP float64
		if _, err := fmt.Sscanf(topPStr, "%f", &topP); err == nil {
			return &topP
		}
	}
	return nil
}

func GetTopK() *int {
	topKStr := os.Getenv("OPENROUTER_TOP_K")
	if topKStr != "" {
		var topK int
		if _, err := fmt.Sscanf(topKStr, "%d", &topK); err == nil {
			return &topK
		}
	}
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
func ChatWithHistory(messages []Message, customSystemPrompt string) (string, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	model := GetModel()
	log.Printf("[LLM] Calling OpenRouter API with model: %s, message history count: %d", model, len(messages))

	messagesWithHistory := buildMessagesWithHistory(messages, customSystemPrompt)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messagesWithHistory,
		Stream:      false,
		Temperature: GetTemperature(),
		TopP:        GetTopP(),
		TopK:        GetTopK(),
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
func ChatWithHistoryStream(messages []Message, customSystemPrompt string) (<-chan string, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	model := GetModel()
	log.Printf("[LLM] Calling OpenRouter API (streaming) with model: %s, message history count: %d", model, len(messages))

	messagesWithHistory := buildMessagesWithHistory(messages, customSystemPrompt)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messagesWithHistory,
		Stream:      true,
		Temperature: GetTemperature(),
		TopP:        GetTopP(),
		TopK:        GetTopK(),
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
