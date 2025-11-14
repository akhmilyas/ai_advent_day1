package handlers

import (
	"chat-app/internal/app"
	"chat-app/internal/auth"
	"chat-app/internal/config"
	"chat-app/internal/context"
	"chat-app/internal/db"
	"chat-app/internal/llm"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type ChatRequest struct {
	Message             string        `json:"message,omitempty"`
	Messages            []llm.Message `json:"messages,omitempty"`
	ConversationID      string        `json:"conversation_id,omitempty"`
	SystemPrompt        string        `json:"system_prompt,omitempty"`
	ResponseFormat      string        `json:"response_format,omitempty"`
	ResponseSchema      string        `json:"response_schema,omitempty"`
	Model               string        `json:"model,omitempty"`
	Temperature         *float64      `json:"temperature,omitempty"`
	Provider            string        `json:"provider,omitempty"`               // "openrouter" or "genkit"
	UseWarAndPeace      bool          `json:"use_war_and_peace,omitempty"`      // Append War and Peace to system prompt
	WarAndPeacePercent  int           `json:"war_and_peace_percent,omitempty"`  // Percentage of War and Peace to include (1-100)
}

type ChatResponse struct {
	Response       string `json:"response"`
	ConversationID string `json:"conversation_id,omitempty"`
	Model          string `json:"model,omitempty"`
	Error          string `json:"error,omitempty"`
}

type ConversationInfo struct {
	ID                      string  `json:"id"`
	Title                   string  `json:"title"`
	ResponseFormat          string  `json:"response_format"`
	ResponseSchema          string  `json:"response_schema"`
	SummarizedUpToMessageID *string `json:"summarized_up_to_message_id,omitempty"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
}

type ConversationsResponse struct {
	Conversations []ConversationInfo `json:"conversations"`
}

type MessageData struct {
	ID               string   `json:"id"`
	Role             string   `json:"role"`
	Content          string   `json:"content"`
	Model            string   `json:"model,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	PromptTokens     *int     `json:"prompt_tokens,omitempty"`
	CompletionTokens *int     `json:"completion_tokens,omitempty"`
	TotalTokens      *int     `json:"total_tokens,omitempty"`
	TotalCost        *float64 `json:"total_cost,omitempty"`
	Latency          *int     `json:"latency,omitempty"`
	GenerationTime   *int     `json:"generation_time,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

type MessagesResponse struct {
	Messages []MessageData `json:"messages"`
}

type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ModelsResponse struct {
	Models []config.Model `json:"models"`
}

type SummarizeRequest struct {
	Model       string   `json:"model,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

type SummarizeResponse struct {
	Summary              string `json:"summary"`
	SummarizedUpToMsgID  string `json:"summarized_up_to_message_id,omitempty"`
	ConversationID       string `json:"conversation_id"`
	Error                string `json:"error,omitempty"`
}

type SummaryData struct {
	ID                      string `json:"id"`
	SummaryContent          string `json:"summary_content"`
	SummarizedUpToMessageID string `json:"summarized_up_to_message_id"`
	UsageCount              int    `json:"usage_count"`
	CreatedAt               string `json:"created_at"`
}

type SummariesResponse struct {
	Summaries []SummaryData `json:"summaries"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ChatHandlers struct{
	config *app.Config
}

func NewChatHandlers(config *app.Config) *ChatHandlers {
	return &ChatHandlers{
		config: config,
	}
}

// Helper functions for common operations

// sendError sends a standardized JSON error response
func (ch *ChatHandlers) sendError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errResp := ErrorResponse{
		Code:    status,
		Message: message,
	}
	if err != nil {
		errResp.Error = err.Error()
	}
	json.NewEncoder(w).Encode(errResp)
}

// getUserFromContext extracts and validates user from request context
func (ch *ChatHandlers) getUserFromContext(r *http.Request) (*db.User, error) {
	username := r.Context().Value(auth.UserContextKey).(string)
	return ch.config.DB.GetUserByUsername(username)
}

// getOrCreateConversation retrieves an existing conversation or creates a new one
func (ch *ChatHandlers) getOrCreateConversation(req *ChatRequest, userID string) (*db.Conversation, error) {
	if req.ConversationID != "" {
		return ch.config.DB.GetConversation(req.ConversationID)
	}

	// Create new conversation with first message as title and specified format
	title := req.Message
	// Use rune slicing to avoid cutting multi-byte UTF-8 characters
	runes := []rune(title)
	if len(runes) > 100 {
		title = string(runes[:100])
	}
	return ch.config.DB.CreateConversation(userID, title, req.ResponseFormat, req.ResponseSchema)
}

// validateConversationOwnership checks if the user owns the conversation
func (ch *ChatHandlers) validateConversationOwnership(conversation *db.Conversation, userID string) error {
	if conversation.UserID != userID {
		return fmt.Errorf("unauthorized: user does not own this conversation")
	}
	return nil
}

// validateModel checks if the provided model ID is valid
func (ch *ChatHandlers) validateModel(modelID string) error {
	if modelID != "" && !ch.config.ModelsConfig.IsValidModel(modelID) {
		return fmt.Errorf("invalid model specified")
	}
	return nil
}

// getConversationHistory retrieves the conversation history considering summaries
func (ch *ChatHandlers) getConversationHistory(conversationID string) ([]llm.Message, *db.ConversationSummary, error) {
	// Check if there's an active summary for this conversation
	activeSummary, err := ch.config.DB.GetActiveSummary(conversationID)
	var currentHistory []llm.Message

	if err == nil && activeSummary != nil {
		// Active summary exists - use it instead of full history
		log.Printf("[CHAT] Using active summary (usage count: %d)", activeSummary.UsageCount)

		// Get messages after the summarized point
		if activeSummary.SummarizedUpToMessageID != nil {
			newMessages, err := ch.config.DB.GetMessagesAfterMessage(conversationID, *activeSummary.SummarizedUpToMessageID)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting messages after summary: %w", err)
			}
			currentHistory = newMessages
			log.Printf("[CHAT] Using summary + %d new messages", len(newMessages))
		} else {
			// No messages after summary (shouldn't happen, but handle gracefully)
			currentHistory = []llm.Message{}
			log.Printf("[CHAT] Using summary with no new messages")
		}

		// Increment summary usage count
		if err := ch.config.DB.IncrementSummaryUsageCount(activeSummary.ID); err != nil {
			log.Printf("[CHAT] Warning: failed to increment summary usage count: %v", err)
		}

		return currentHistory, activeSummary, nil
	}

	// No active summary - use full conversation history
	currentHistory, err = ch.config.DB.GetConversationMessages(conversationID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting conversation history: %w", err)
	}
	log.Printf("[CHAT] Using full conversation history: %d messages", len(currentHistory))

	return currentHistory, nil, nil
}

// buildSystemPrompt builds the effective system prompt based on conversation format and summary
func (ch *ChatHandlers) buildSystemPrompt(conversation *db.Conversation, activeSummary *db.ConversationSummary, customPrompt string, useWarAndPeace bool, warAndPeacePercent int) string {
	var effectiveSystemPrompt string

	// Add summary context if available
	summaryContext := ""
	if activeSummary != nil {
		summaryContext = fmt.Sprintf("Previous conversation summary:\n%s\n\n", activeSummary.SummaryContent)
		log.Printf("[CHAT] Using summary as context with user prompt")
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
		effectiveSystemPrompt = ch.appendWarAndPeaceContext(effectiveSystemPrompt, warAndPeacePercent)
	}

	return effectiveSystemPrompt
}

// appendWarAndPeaceContext appends War and Peace text to the system prompt
func (ch *ChatHandlers) appendWarAndPeaceContext(systemPrompt string, percent int) string {
	warAndPeaceText := context.GetWarAndPeace()
	if warAndPeaceText == "" {
		log.Printf("[CHAT] Warning: War and Peace text not loaded")
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

	log.Printf("[CHAT] Appended War and Peace context: %d%% (%.2f MB of %.2f MB)",
		percent,
		float64(len(textToAppend))/1024/1024,
		float64(totalChars)/1024/1024)

	return systemPrompt + "\n\nContext (War and Peace by Leo Tolstoy):\n" + textToAppend
}

// ChatHandler is the REST endpoint for chat
func (ch *ChatHandlers) ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	log.Printf("Chat request from user: %s", username)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Message == "" {
		ch.sendError(w, http.StatusBadRequest, "Message cannot be empty", nil)
		return
	}

	log.Printf("[CHAT] User input: %s", req.Message)

	// Get user from database
	user, err := ch.getUserFromContext(r)
	if err != nil {
		log.Printf("[CHAT] Error getting user: %v", err)
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Get or create conversation
	conversation, err := ch.getOrCreateConversation(&req, user.ID)
	if err != nil {
		log.Printf("[CHAT] Error getting/creating conversation: %v", err)
		ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		return
	}

	// Verify user owns this conversation
	if req.ConversationID != "" {
		if err := ch.validateConversationOwnership(conversation, user.ID); err != nil {
			ch.sendError(w, http.StatusForbidden, "Unauthorized", err)
			return
		}
	}

	// Validate model if provided
	if err := ch.validateModel(req.Model); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Invalid model specified", err)
		return
	}

	// Add user message to database (user messages don't have a model, temperature, provider, or usage data)
	if _, err := ch.config.DB.AddMessage(conversation.ID, "user", req.Message, "", nil, "", "", nil, nil, nil, nil, nil, nil); err != nil {
		log.Printf("[CHAT] Error adding user message: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error saving message", err)
		return
	}

	// Get conversation history
	currentHistory, err := ch.config.DB.GetConversationMessages(conversation.ID)
	if err != nil {
		log.Printf("[CHAT] Error getting conversation history: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error retrieving conversation history", err)
		return
	}

	log.Printf("[CHAT] Conversation history length: %d messages", len(currentHistory))

	// Get LLM provider based on request
	provider := llm.GetProviderFromString(req.Provider)
	log.Printf("[CHAT] Using provider: %T", provider)

	// Get response with full conversation history
	response, err := provider.ChatWithHistory(currentHistory, req.SystemPrompt, conversation.ResponseFormat, req.Model, req.Temperature)
	if err != nil {
		log.Printf("[CHAT] Error from LLM: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error from LLM provider", err)
		return
	}

	log.Printf("[CHAT] LLM response: %s", response)

	// Determine which model was actually used
	usedModel := req.Model
	if usedModel == "" {
		usedModel = provider.GetDefaultModel()
	}

	// Add assistant response to database with model, temperature, and provider (no usage data for non-streaming)
	if _, err := ch.config.DB.AddMessage(conversation.ID, "assistant", response, usedModel, req.Temperature, req.Provider, "", nil, nil, nil, nil, nil, nil); err != nil {
		log.Printf("[CHAT] Error adding assistant message: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error saving response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Response:       response,
		ConversationID: conversation.ID,
		Model:          usedModel,
	})
}

// ChatStreamHandler is the SSE endpoint for streaming chat responses
func (ch *ChatHandlers) ChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	log.Printf("Chat stream request from user: %s", username)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Message == "" {
		ch.sendError(w, http.StatusBadRequest, "Message cannot be empty", nil)
		return
	}

	log.Printf("[CHAT] User input (stream): %s", req.Message)

	// Get user from database
	user, err := ch.getUserFromContext(r)
	if err != nil {
		log.Printf("[CHAT] Error getting user: %v", err)
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Get or create conversation
	conversation, err := ch.getOrCreateConversation(&req, user.ID)
	if err != nil {
		log.Printf("[CHAT] Error getting/creating conversation: %v", err)
		ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		return
	}

	// Verify user owns this conversation
	if req.ConversationID != "" {
		if err := ch.validateConversationOwnership(conversation, user.ID); err != nil {
			ch.sendError(w, http.StatusForbidden, "Unauthorized", err)
			return
		}
	}

	// Validate model if provided
	if err := ch.validateModel(req.Model); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Invalid model specified", err)
		return
	}

	// Add user message to database (user messages don't have a model, temperature, provider, or usage data)
	if _, err := ch.config.DB.AddMessage(conversation.ID, "user", req.Message, "", nil, "", "", nil, nil, nil, nil, nil, nil); err != nil {
		log.Printf("[CHAT] Error adding user message: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error saving message", err)
		return
	}

	// Get conversation history (considering summaries)
	currentHistory, activeSummary, err := ch.getConversationHistory(conversation.ID)
	if err != nil {
		log.Printf("[CHAT] %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error retrieving conversation history", err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		ch.sendError(w, http.StatusInternalServerError, "Streaming not supported", nil)
		return
	}

	// Set SSE headers (after flusher check)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Build the system prompt based on conversation's response format (stored in DB)
	effectiveSystemPrompt := ch.buildSystemPrompt(conversation, activeSummary, req.SystemPrompt, req.UseWarAndPeace, req.WarAndPeacePercent)
	log.Printf("[CHAT] Using conversation format: %s", conversation.ResponseFormat)

	// Get LLM provider based on request
	provider := llm.GetProviderFromString(req.Provider)
	log.Printf("[CHAT] Using provider for streaming: %T", provider)

	// Get streaming response from LLM
	chunks, err := provider.ChatWithHistoryStream(currentHistory, effectiveSystemPrompt, conversation.ResponseFormat, req.Model, req.Temperature)
	if err != nil {
		log.Printf("[CHAT] Error from LLM stream: %v", err)
		fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", err.Error())
		return
	}

	// Send conversation ID as first event
	fmt.Fprintf(w, "data: CONV_ID:%s\n\n", conversation.ID)
	flusher.Flush()
	log.Printf("[CHAT] Sent conversation ID: %s", conversation.ID)

	// Determine which model was actually used
	usedModel := req.Model
	if usedModel == "" {
		usedModel = provider.GetDefaultModel()
	}

	// Send model as second event
	fmt.Fprintf(w, "data: MODEL:%s\n\n", usedModel)
	flusher.Flush()
	log.Printf("[CHAT] Sent model: %s", usedModel)

	// Send temperature as third event
	if req.Temperature != nil {
		fmt.Fprintf(w, "data: TEMPERATURE:%.2f\n\n", *req.Temperature)
		flusher.Flush()
		log.Printf("[CHAT] Sent temperature: %.2f", *req.Temperature)
	}

	// Buffer to accumulate the full response and metadata
	var fullResponse string
	var generationID string
	var usage *llm.ResponseUsage

	// Stream chunks to client using SSE format
	for streamChunk := range chunks {
		if streamChunk.Metadata != nil {
			// Capture metadata from final chunk
			if streamChunk.Metadata.GenerationID != "" {
				generationID = streamChunk.Metadata.GenerationID
			}
			if streamChunk.Metadata.Usage != nil {
				usage = streamChunk.Metadata.Usage
			}
		} else if streamChunk.Content != "" {
			// Stream content chunk
			fullResponse += streamChunk.Content
			// Escape newlines in chunk for SSE format
			escapedChunk := strings.ReplaceAll(streamChunk.Content, "\n", "\\n")
			// Send chunk as SSE event
			fmt.Fprintf(w, "data: %s\n\n", escapedChunk)
			flusher.Flush()
			log.Printf("[CHAT] Sent chunk: %q", streamChunk.Content)
		}
	}

	// Fetch cost information from OpenRouter if generation ID is available
	var totalCost *float64
	var promptTokens, completionTokens, totalTokens *int
	var latency, generationTime *int

	if generationID != "" {
		log.Printf("[CHAT] Fetching generation cost for ID: %s", generationID)
		if genData, err := provider.FetchGenerationCost(generationID); err == nil {
			totalCost = &genData.TotalCost
			// Use native tokens instead of regular tokens
			promptTokens = &genData.NativeTokensPrompt
			completionTokens = &genData.NativeTokensCompletion
			totalTokensVal := genData.NativeTokensPrompt + genData.NativeTokensCompletion
			totalTokens = &totalTokensVal
			latency = &genData.Latency
			generationTime = &genData.GenerationTime

			// Send usage data via SSE
			fmt.Fprintf(w, "data: USAGE:{\"prompt_tokens\":%d,\"completion_tokens\":%d,\"total_tokens\":%d,\"total_cost\":%.6f,\"latency\":%d,\"generation_time\":%d}\n\n",
				*promptTokens, *completionTokens, *totalTokens, *totalCost, *latency, *generationTime)
			flusher.Flush()
			log.Printf("[CHAT] Sent usage data: tokens=%d, cost=$%.6f, latency=%dms, generation_time=%dms", *totalTokens, *totalCost, *latency, *generationTime)
		} else {
			log.Printf("[CHAT] Error fetching generation cost: %v", err)
			// Fallback to usage data from streaming response if available
			if usage != nil {
				promptTokens = &usage.PromptTokens
				completionTokens = &usage.CompletionTokens
				totalTokens = &usage.TotalTokens

				// Send usage data without cost via SSE
				fmt.Fprintf(w, "data: USAGE:{\"prompt_tokens\":%d,\"completion_tokens\":%d,\"total_tokens\":%d}\n\n",
					*promptTokens, *completionTokens, *totalTokens)
				flusher.Flush()
				log.Printf("[CHAT] Sent usage data (no cost): tokens=%d", *totalTokens)
			}
		}
	} else if usage != nil {
		// No generation ID but have usage from stream
		promptTokens = &usage.PromptTokens
		completionTokens = &usage.CompletionTokens
		totalTokens = &usage.TotalTokens

		// Send usage data without cost via SSE
		fmt.Fprintf(w, "data: USAGE:{\"prompt_tokens\":%d,\"completion_tokens\":%d,\"total_tokens\":%d}\n\n",
			*promptTokens, *completionTokens, *totalTokens)
		flusher.Flush()
		log.Printf("[CHAT] Sent usage data (no cost): tokens=%d", *totalTokens)
	}

	// Add assistant response to database after streaming completes
	if fullResponse != "" {
		if _, err := ch.config.DB.AddMessage(conversation.ID, "assistant", fullResponse, usedModel, req.Temperature, req.Provider,
			generationID, promptTokens, completionTokens, totalTokens, totalCost, latency, generationTime); err != nil {
			log.Printf("[CHAT] Error adding assistant message: %v", err)
		}
		log.Printf("[CHAT] Full LLM response: %s", fullResponse)
	}

	// Send completion marker
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// GetConversationsHandler returns all conversations for the authenticated user
func (ch *ChatHandlers) GetConversationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	log.Printf("Get conversations request from user: %s", username)

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[CHAT] Error getting user: %v", err)
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Get all conversations for user
	conversations, err := ch.config.DB.GetConversationsByUser(user.ID)
	if err != nil {
		log.Printf("[CHAT] Error getting conversations: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error retrieving conversations", err)
		return
	}

	// Convert to response format
	convInfos := make([]ConversationInfo, 0, len(conversations))
	for _, conv := range conversations {
		// Get active summary for this conversation if it exists
		var summarizedUpToMsgID *string
		if summary, err := ch.config.DB.GetActiveSummary(conv.ID); err == nil && summary != nil {
			summarizedUpToMsgID = summary.SummarizedUpToMessageID
		}

		convInfos = append(convInfos, ConversationInfo{
			ID:                      conv.ID,
			Title:                   conv.Title,
			ResponseFormat:          conv.ResponseFormat,
			ResponseSchema:          conv.ResponseSchema,
			SummarizedUpToMessageID: summarizedUpToMsgID,
			CreatedAt:               conv.CreatedAt.String(),
			UpdatedAt:               conv.UpdatedAt.String(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConversationsResponse{
		Conversations: convInfos,
	})
}

// GetConversationMessagesHandler returns all messages from a specific conversation
func (ch *ChatHandlers) GetConversationMessagesHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(auth.UserContextKey).(string)
	convID := r.PathValue("id")
	log.Printf("Get conversation messages request from user: %s for conversation: %s", username, convID)

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[CHAT] Error getting user: %v", err)
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Get conversation and verify ownership
	conversation, err := ch.config.DB.GetConversation(convID)
	if err != nil {
		log.Printf("[CHAT] Error getting conversation: %v", err)
		ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		return
	}

	// Verify user owns this conversation
	if conversation.UserID != user.ID {
		ch.sendError(w, http.StatusForbidden, "Unauthorized", nil)
		return
	}

	// Get messages for conversation
	messages, err := ch.config.DB.GetConversationMessagesWithDetails(convID)
	if err != nil {
		log.Printf("[CHAT] Error getting messages: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error retrieving messages", err)
		return
	}

	// Convert to response format
	msgData := make([]MessageData, 0, len(messages))
	for _, msg := range messages {
		msgData = append(msgData, MessageData{
			ID:               msg.ID,
			Role:             msg.Role,
			Content:          msg.Content,
			Model:            msg.Model,
			Temperature:      msg.Temperature,
			PromptTokens:     msg.PromptTokens,
			CompletionTokens: msg.CompletionTokens,
			TotalTokens:      msg.TotalTokens,
			TotalCost:        msg.TotalCost,
			Latency:          msg.Latency,
			GenerationTime:   msg.GenerationTime,
			CreatedAt:        msg.CreatedAt.String(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MessagesResponse{
		Messages: msgData,
	})
}

// DeleteConversationHandler deletes a specific conversation
func (ch *ChatHandlers) DeleteConversationHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(auth.UserContextKey).(string)
	convID := r.PathValue("id")
	log.Printf("Delete conversation request from user: %s for conversation: %s", username, convID)

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[CHAT] Error getting user: %v", err)
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Get conversation and verify ownership
	conversation, err := ch.config.DB.GetConversation(convID)
	if err != nil {
		log.Printf("[CHAT] Error getting conversation: %v", err)
		ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		return
	}

	// Verify user owns this conversation
	if conversation.UserID != user.ID {
		ch.sendError(w, http.StatusForbidden, "Unauthorized", nil)
		return
	}

	// Delete the conversation
	if err := ch.config.DB.DeleteConversation(convID); err != nil {
		log.Printf("[CHAT] Error deleting conversation: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error deleting conversation", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DeleteResponse{
		Success: true,
		Message: "Conversation deleted successfully",
	})
}

// GetModelsHandler returns the list of available models
func (ch *ChatHandlers) GetModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	models := ch.config.ModelsConfig.GetAvailableModels()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ModelsResponse{
		Models: models,
	})
}

// SummarizeConversationHandler creates a summary of the conversation
func (ch *ChatHandlers) SummarizeConversationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	convID := r.PathValue("id")
	log.Printf("Summarize conversation request from user: %s for conversation: %s", username, convID)

	var req SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is acceptable, use defaults
		req.Model = ""
		req.Temperature = nil
	}

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[SUMMARIZE] Error getting user: %v", err)
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Get conversation and verify ownership
	conversation, err := ch.config.DB.GetConversation(convID)
	if err != nil {
		log.Printf("[SUMMARIZE] Error getting conversation: %v", err)
		ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		return
	}

	// Verify user owns this conversation
	if conversation.UserID != user.ID {
		ch.sendError(w, http.StatusForbidden, "Unauthorized", nil)
		return
	}

	// Check if there's an existing active summary
	activeSummary, err := ch.config.DB.GetActiveSummary(convID)
	var messagesToSummarize []llm.Message
	var lastMessageID *string

	if err != nil || activeSummary == nil {
		// No active summary exists - summarize all messages
		log.Printf("[SUMMARIZE] No active summary found, summarizing all messages")
		messagesToSummarize, err = ch.config.DB.GetConversationMessages(convID)
		if err != nil {
			log.Printf("[SUMMARIZE] Error getting conversation messages: %v", err)
			ch.sendError(w, http.StatusInternalServerError, "Error retrieving messages", err)
			return
		}

		// Get the last message ID
		lastMessageID, err = ch.config.DB.GetLastMessageID(convID)
		if err != nil {
			log.Printf("[SUMMARIZE] Error getting last message ID: %v", err)
			ch.sendError(w, http.StatusInternalServerError, "Error retrieving last message", err)
			return
		}
	} else if activeSummary.UsageCount >= 2 {
		// Summary has been used 2+ times - create new summary from old summary + new messages
		log.Printf("[SUMMARIZE] Active summary used %d times, creating new summary", activeSummary.UsageCount)

		// Start with the old summary as a "system" message
		messagesToSummarize = []llm.Message{
			{Role: "assistant", Content: fmt.Sprintf("Previous summary:\n%s", activeSummary.SummaryContent)},
		}

		// Get messages after the last summarized message
		if activeSummary.SummarizedUpToMessageID != nil {
			newMessages, err := ch.config.DB.GetMessagesAfterMessage(convID, *activeSummary.SummarizedUpToMessageID)
			if err != nil {
				log.Printf("[SUMMARIZE] Error getting messages after last summarized: %v", err)
				ch.sendError(w, http.StatusInternalServerError, "Error retrieving new messages", err)
				return
			}
			messagesToSummarize = append(messagesToSummarize, newMessages...)
		}

		// Get the last message ID
		lastMessageID, err = ch.config.DB.GetLastMessageID(convID)
		if err != nil {
			log.Printf("[SUMMARIZE] Error getting last message ID: %v", err)
			ch.sendError(w, http.StatusInternalServerError, "Error retrieving last message", err)
			return
		}
	} else {
		// Summary exists but hasn't been used enough yet - don't create new summary
		log.Printf("[SUMMARIZE] Active summary exists with usage count %d, not creating new summary", activeSummary.UsageCount)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SummarizeResponse{
			Summary:             activeSummary.SummaryContent,
			SummarizedUpToMsgID: *activeSummary.SummarizedUpToMessageID,
			ConversationID:      convID,
		})
		return
	}

	// Validate model if provided
	if err := ch.validateModel(req.Model); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Invalid model specified", err)
		return
	}

	// Get LLM provider (always use openrouter for summarization)
	provider := llm.NewOpenRouterProvider()

	// Build summarization system prompt
	summarizationPrompt := os.Getenv("OPENROUTER_SUMMARIZATION_PROMPT")
	if summarizationPrompt == "" {
		summarizationPrompt = `You are a conversation summarizer. Your task is to create a concise, comprehensive summary of the conversation that captures:
1. The main topics discussed
2. Key questions asked and answers provided
3. Important decisions or conclusions reached
4. Any action items or next steps mentioned

Format the summary in a clear, structured way that can be used as context for continuing the conversation. Keep the summary focused and avoid unnecessary details while preserving essential information.`
	}

	// Add the original system prompt from conversation format if it exists
	if conversation.ResponseFormat == "text" {
		// For text format, we should preserve any custom system prompt that was used
		// However, we don't store the original system prompt, so we'll just use the summarization prompt
		summarizationPrompt = summarizationPrompt
	}

	// Call LLM to generate summary (using ChatForSummarization to avoid default system prompt)
	log.Printf("[SUMMARIZE] Calling LLM to generate summary with %d messages", len(messagesToSummarize))
	summaryContent, err := provider.ChatForSummarization(messagesToSummarize, summarizationPrompt, req.Model, req.Temperature)
	if err != nil {
		log.Printf("[SUMMARIZE] Error from LLM: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error generating summary", err)
		return
	}

	log.Printf("[SUMMARIZE] Generated summary: %s", summaryContent)

	// Create new summary in database
	summary, err := ch.config.DB.CreateSummary(convID, summaryContent, lastMessageID)
	if err != nil {
		log.Printf("[SUMMARIZE] Error creating summary: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error saving summary", err)
		return
	}

	// Update conversation to use this new summary
	if err := ch.config.DB.UpdateConversationActiveSummary(convID, summary.ID); err != nil {
		log.Printf("[SUMMARIZE] Error updating active summary: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error updating conversation", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SummarizeResponse{
		Summary:             summaryContent,
		SummarizedUpToMsgID: *lastMessageID,
		ConversationID:      convID,
	})
}

// GetConversationSummariesHandler retrieves all summaries for a conversation
func (ch *ChatHandlers) GetConversationSummariesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	convID := r.PathValue("id")
	log.Printf("Get summaries request from user: %s for conversation: %s", username, convID)

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[SUMMARIES] Error getting user: %v", err)
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Get conversation and verify ownership
	conversation, err := ch.config.DB.GetConversation(convID)
	if err != nil {
		log.Printf("[SUMMARIES] Error getting conversation: %v", err)
		ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		return
	}

	// Verify user owns this conversation
	if conversation.UserID != user.ID {
		ch.sendError(w, http.StatusForbidden, "Unauthorized", nil)
		return
	}

	// Get all summaries for conversation
	summaries, err := ch.config.DB.GetAllSummaries(convID)
	if err != nil {
		log.Printf("[SUMMARIES] Error getting summaries: %v", err)
		ch.sendError(w, http.StatusInternalServerError, "Error retrieving summaries", err)
		return
	}

	// Convert to response format
	summaryData := make([]SummaryData, 0, len(summaries))
	for _, summary := range summaries {
		upToMsgID := ""
		if summary.SummarizedUpToMessageID != nil {
			upToMsgID = *summary.SummarizedUpToMessageID
		}
		summaryData = append(summaryData, SummaryData{
			ID:                      summary.ID,
			SummaryContent:          summary.SummaryContent,
			SummarizedUpToMessageID: upToMsgID,
			UsageCount:              summary.UsageCount,
			CreatedAt:               summary.CreatedAt.String(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SummariesResponse{
		Summaries: summaryData,
	})
}
