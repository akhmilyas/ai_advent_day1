package chat

import (
	"chat-app/internal/app"
	"chat-app/internal/context"
	"chat-app/internal/logger"
	"chat-app/internal/repository/db"
	"chat-app/internal/service/llm"
	"fmt"

	"github.com/sirupsen/logrus"
)

// SendMessageRequest contains all the parameters needed to send a message
type SendMessageRequest struct {
	Message            string
	ConversationID     string
	SystemPrompt       string
	ResponseFormat     string
	ResponseSchema     string
	Model              string
	Temperature        *float64
	Provider           string
	UseWarAndPeace     bool
	WarAndPeacePercent int
	UserID             string // Extracted from auth context
}

// SendMessageResponse contains the response from sending a message
type SendMessageResponse struct {
	Response       string
	ConversationID string
	Model          string
	GenerationID   string
	Usage          *llm.ResponseUsage
}

// ChatService handles the business logic for chat operations
type ChatService struct {
	db          db.Database
	config      *app.Config
	llmProvider llm.LLMProvider
}

// NewChatService creates a new ChatService
func NewChatService(database db.Database, config *app.Config) *ChatService {
	// Create LLM provider with configuration
	llmProvider := llm.NewOpenRouterProvider(
		&config.AppConfig.LLM,
		config.AppConfig.Models,
	)

	return &ChatService{
		db:          database,
		config:      config,
		llmProvider: llmProvider,
	}
}

// SendMessage processes a chat message and returns the LLM response
func (s *ChatService) SendMessage(req SendMessageRequest) (*SendMessageResponse, error) {
	// Get or create conversation
	conversation, err := s.getOrCreateConversation(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create conversation: %w", err)
	}

	// Verify user owns this conversation
	if req.ConversationID != "" && conversation.UserID != req.UserID {
		return nil, fmt.Errorf("unauthorized: user does not own this conversation")
	}

	// Validate model if provided
	if err := s.validateModel(req.Model); err != nil {
		return nil, fmt.Errorf("invalid model: %w", err)
	}

	// Add user message to database (user messages don't have model, temperature, provider, or usage data)
	if _, err := s.db.AddMessage(conversation.ID, "user", req.Message, "", nil, "", "", nil, nil, nil, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Get conversation history (considering summaries)
	currentHistory, activeSummary, err := s.getConversationHistory(conversation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conversation history: %w", err)
	}

	// Build the system prompt based on conversation's response format
	effectiveSystemPrompt := s.buildSystemPrompt(conversation, activeSummary, req.SystemPrompt, req.UseWarAndPeace, req.WarAndPeacePercent)

	logger.Log.WithFields(logrus.Fields{
		"conversation_id": conversation.ID,
		"message_count":   len(currentHistory),
		"format":          conversation.ResponseFormat,
	}).Debug("Prepared for LLM call")

	// Get response from LLM (non-streaming)
	response, err := s.llmProvider.ChatWithHistory(currentHistory, effectiveSystemPrompt, conversation.ResponseFormat, req.Model, req.Temperature)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	// Determine which model was actually used
	usedModel := req.Model
	if usedModel == "" {
		usedModel = s.llmProvider.GetDefaultModel()
	}

	// Add assistant response to database (no usage data for non-streaming)
	if _, err := s.db.AddMessage(conversation.ID, "assistant", response, usedModel, req.Temperature, req.Provider, "", nil, nil, nil, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	return &SendMessageResponse{
		Response:       response,
		ConversationID: conversation.ID,
		Model:          usedModel,
	}, nil
}

// StreamMessageChunk represents a chunk of streaming response
type StreamMessageChunk struct {
	Content      string
	Metadata     *llm.StreamMetadata
	ConvID       string // Set on first chunk
	Model        string // Set on first chunk
	Temperature  *float64 // Set on first chunk
}

// SendMessageStream processes a chat message and streams the LLM response
func (s *ChatService) SendMessageStream(req SendMessageRequest) (<-chan StreamMessageChunk, error) {
	// Get or create conversation
	conversation, err := s.getOrCreateConversation(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create conversation: %w", err)
	}

	// Verify user owns this conversation
	if req.ConversationID != "" && conversation.UserID != req.UserID {
		return nil, fmt.Errorf("unauthorized: user does not own this conversation")
	}

	// Validate model if provided
	if err := s.validateModel(req.Model); err != nil {
		return nil, fmt.Errorf("invalid model: %w", err)
	}

	// Add user message to database
	if _, err := s.db.AddMessage(conversation.ID, "user", req.Message, "", nil, "", "", nil, nil, nil, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Get conversation history (considering summaries)
	currentHistory, activeSummary, err := s.getConversationHistory(conversation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conversation history: %w", err)
	}

	// Build the system prompt based on conversation's response format
	effectiveSystemPrompt := s.buildSystemPrompt(conversation, activeSummary, req.SystemPrompt, req.UseWarAndPeace, req.WarAndPeacePercent)

	// Determine which model will be used
	usedModel := req.Model
	if usedModel == "" {
		usedModel = s.llmProvider.GetDefaultModel()
	}

	logger.Log.WithFields(logrus.Fields{
		"conversation_id": conversation.ID,
		"message_count":   len(currentHistory),
		"format":          conversation.ResponseFormat,
		"model":           usedModel,
	}).Debug("Starting streaming LLM call")

	// Get streaming response from LLM
	llmChunks, err := s.llmProvider.ChatWithHistoryStream(currentHistory, effectiveSystemPrompt, conversation.ResponseFormat, req.Model, req.Temperature)
	if err != nil {
		return nil, fmt.Errorf("LLM streaming error: %w", err)
	}

	// Create output channel
	outputChan := make(chan StreamMessageChunk)

	// Process streaming chunks in a goroutine
	go func() {
		defer close(outputChan)

		var fullResponse string
		var generationID string
		var usage *llm.ResponseUsage
		firstChunk := true

		for chunk := range llmChunks {
			if chunk.Metadata != nil {
				// Capture metadata from final chunk
				if chunk.Metadata.GenerationID != "" {
					generationID = chunk.Metadata.GenerationID
				}
				if chunk.Metadata.Usage != nil {
					usage = chunk.Metadata.Usage
				}
			} else if chunk.Content != "" {
				// Send metadata on first content chunk
				if firstChunk {
					outputChan <- StreamMessageChunk{
						ConvID:      conversation.ID,
						Model:       usedModel,
						Temperature: req.Temperature,
					}
					firstChunk = false
				}

				// Accumulate and stream content
				fullResponse += chunk.Content
				outputChan <- StreamMessageChunk{
					Content: chunk.Content,
				}
			}
		}

		// Fetch cost information if generation ID is available
		var totalCost *float64
		var promptTokens, completionTokens, totalTokens *int
		var latency, generationTime *int

		if generationID != "" {
			logger.Log.WithField("generation_id", generationID).Debug("Fetching generation cost")
			if genData, err := s.llmProvider.FetchGenerationCost(generationID); err == nil {
				totalCost = &genData.TotalCost
				promptTokens = &genData.NativeTokensPrompt
				completionTokens = &genData.NativeTokensCompletion
				totalTokensVal := genData.NativeTokensPrompt + genData.NativeTokensCompletion
				totalTokens = &totalTokensVal
				latency = &genData.Latency
				generationTime = &genData.GenerationTime

				logger.Log.WithFields(logrus.Fields{
					"total_tokens":      *totalTokens,
					"cost":              *totalCost,
					"latency_ms":        *latency,
					"generation_time_ms": *generationTime,
				}).Debug("Fetched generation cost")
			} else {
				logger.Log.WithError(err).Warn("Error fetching generation cost")
				// Fallback to usage from streaming
				if usage != nil {
					promptTokens = &usage.PromptTokens
					completionTokens = &usage.CompletionTokens
					totalTokens = &usage.TotalTokens
				}
			}
		} else if usage != nil {
			// No generation ID but have usage from stream
			promptTokens = &usage.PromptTokens
			completionTokens = &usage.CompletionTokens
			totalTokens = &usage.TotalTokens
		}

		// Send final metadata chunk with usage info
		if promptTokens != nil || totalCost != nil {
			metadata := &llm.StreamMetadata{
				GenerationID:   generationID,
				TotalCost:      totalCost,
				Latency:        latency,
				GenerationTime: generationTime,
			}
			if promptTokens != nil {
				metadata.Usage = &llm.ResponseUsage{
					PromptTokens:     *promptTokens,
					CompletionTokens: *completionTokens,
					TotalTokens:      *totalTokens,
				}
			}
			outputChan <- StreamMessageChunk{
				Metadata: metadata,
			}
		}

		// Save assistant response to database
		if fullResponse != "" {
			if _, err := s.db.AddMessage(conversation.ID, "assistant", fullResponse, usedModel, req.Temperature, req.Provider,
				generationID, promptTokens, completionTokens, totalTokens, totalCost, latency, generationTime); err != nil {
				logger.Log.WithError(err).Error("Error adding assistant message")
			}
			logger.Log.WithField("response_chars", len(fullResponse)).Debug("Completed streaming response")
		}
	}()

	return outputChan, nil
}

// getOrCreateConversation retrieves an existing conversation or creates a new one
func (s *ChatService) getOrCreateConversation(req SendMessageRequest) (*db.Conversation, error) {
	if req.ConversationID != "" {
		return s.db.GetConversation(req.ConversationID)
	}

	// Create new conversation with first message as title and specified format
	title := req.Message
	runes := []rune(title)
	if len(runes) > 100 {
		title = string(runes[:100])
	}
	return s.db.CreateConversation(req.UserID, title, req.ResponseFormat, req.ResponseSchema)
}

// validateModel checks if the provided model ID is valid
func (s *ChatService) validateModel(modelID string) error {
	if modelID != "" && !s.config.ModelsConfig().IsValidModel(modelID) {
		return fmt.Errorf("invalid model specified")
	}
	return nil
}

// getConversationHistory retrieves the conversation history considering summaries
func (s *ChatService) getConversationHistory(conversationID string) ([]llm.Message, *db.ConversationSummary, error) {
	// Check if there's an active summary for this conversation
	activeSummary, err := s.db.GetActiveSummary(conversationID)
	var currentHistory []llm.Message

	if err == nil && activeSummary != nil {
		// Active summary exists - use it instead of full history
		logger.Log.WithField("usage_count", activeSummary.UsageCount).Info("Using active summary")

		// Get messages after the summarized point
		if activeSummary.SummarizedUpToMessageID != nil {
			newMessages, err := s.db.GetMessagesAfterMessage(conversationID, *activeSummary.SummarizedUpToMessageID)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting messages after summary: %w", err)
			}
			currentHistory = newMessages
			logger.Log.WithField("new_message_count", len(newMessages)).Debug("Using summary with new messages")
		} else {
			// No messages after summary (shouldn't happen, but handle gracefully)
			currentHistory = []llm.Message{}
			logger.Log.Debug("Using summary with no new messages")
		}

		// Increment summary usage count
		if err := s.db.IncrementSummaryUsageCount(activeSummary.ID); err != nil {
			logger.Log.WithError(err).Warn("Failed to increment summary usage count")
		}

		return currentHistory, activeSummary, nil
	}

	// No active summary - use full conversation history
	currentHistory, err = s.db.GetConversationMessages(conversationID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting conversation history: %w", err)
	}
	logger.Log.WithField("message_count", len(currentHistory)).Debug("Using full conversation history")

	return currentHistory, nil, nil
}

// buildSystemPrompt builds the effective system prompt based on conversation format and summary
func (s *ChatService) buildSystemPrompt(conversation *db.Conversation, activeSummary *db.ConversationSummary, customPrompt string, useWarAndPeace bool, warAndPeacePercent int) string {
	var effectiveSystemPrompt string

	// Add summary context if available
	summaryContext := ""
	if activeSummary != nil {
		summaryContext = fmt.Sprintf("Previous conversation summary:\n%s\n\n", activeSummary.SummaryContent)
		logger.Log.Debug("Using summary as context with user prompt")
	}

	// Build format-specific system prompt
	if conversation.ResponseFormat == "json" && conversation.ResponseSchema != "" {
		effectiveSystemPrompt = summaryContext + fmt.Sprintf("You must respond ONLY with valid JSON that matches this exact schema. Do not include any explanatory text, markdown formatting, or code blocks - just the raw JSON.\n\nSchema:\n%s\n\nRemember: Your entire response must be valid JSON matching this schema.", conversation.ResponseSchema)
	} else if conversation.ResponseFormat == "xml" && conversation.ResponseSchema != "" {
		effectiveSystemPrompt = summaryContext + fmt.Sprintf("You must respond ONLY with valid XML that matches this exact schema. Do not include any explanatory text, markdown formatting, or code blocks - just the raw XML.\n\nSchema:\n%s\n\nRemember: Your entire response must be valid XML matching this schema.", conversation.ResponseSchema)
	} else {
		// For text format, combine summary with user's custom system prompt
		effectiveSystemPrompt = summaryContext + customPrompt
	}

	// Append War and Peace context if requested
	if useWarAndPeace {
		effectiveSystemPrompt = s.appendWarAndPeaceContext(effectiveSystemPrompt, warAndPeacePercent)
	}

	return effectiveSystemPrompt
}

// appendWarAndPeaceContext appends War and Peace text to the system prompt
func (s *ChatService) appendWarAndPeaceContext(systemPrompt string, percent int) string {
	warAndPeaceText := context.GetWarAndPeace()
	if warAndPeaceText == "" {
		logger.Log.Warn("War and Peace text not loaded")
		return systemPrompt
	}

	// Calculate how much of the text to include based on percentage
	if percent <= 0 || percent > 100 {
		percent = 100 // Default to 100% if invalid
	}

	// Calculate the number of characters to include
	totalChars := len(warAndPeaceText)
	charsToInclude := (totalChars * percent) / 100

	// Get the substring from the beginning
	textToAppend := warAndPeaceText[:charsToInclude]

	logger.Log.WithFields(logrus.Fields{
		"percent":  percent,
		"size_mb":  float64(len(textToAppend)) / 1024 / 1024,
		"total_mb": float64(totalChars) / 1024 / 1024,
	}).Info("Appended War and Peace context")

	return systemPrompt + "\n\nContext (War and Peace by Leo Tolstoy):\n" + textToAppend
}
