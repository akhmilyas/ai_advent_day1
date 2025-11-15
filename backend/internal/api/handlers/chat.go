package handlers

import (
	"chat-app/internal/app"
	"chat-app/internal/config"
	"chat-app/internal/logger"
	"chat-app/internal/repository/db"
	chatService "chat-app/internal/service/chat"
	conversationService "chat-app/internal/service/conversation"
	"chat-app/internal/service/llm"
	summaryService "chat-app/internal/service/summary"
	"chat-app/pkg/validation"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// Request/Response types

type ChatRequest struct {
	Message            string        `json:"message,omitempty"`
	Messages           []llm.Message `json:"messages,omitempty"`
	ConversationID     string        `json:"conversation_id,omitempty"`
	SystemPrompt       string        `json:"system_prompt,omitempty"`
	ResponseFormat     string        `json:"response_format,omitempty"`
	ResponseSchema     string        `json:"response_schema,omitempty"`
	Model              string        `json:"model,omitempty"`
	Temperature        *float64      `json:"temperature,omitempty"`
	Provider           string        `json:"provider,omitempty"`              // "openrouter" or "genkit"
	UseWarAndPeace     bool          `json:"use_war_and_peace,omitempty"`     // Append War and Peace to system prompt
	WarAndPeacePercent int           `json:"war_and_peace_percent,omitempty"` // Percentage of War and Peace to include (1-100)
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
	Summary             string `json:"summary"`
	SummarizedUpToMsgID string `json:"summarized_up_to_message_id,omitempty"`
	ConversationID      string `json:"conversation_id"`
	Error               string `json:"error,omitempty"`
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

// ChatHandlers uses the service layer for better separation of concerns
type ChatHandlers struct {
	config              *app.Config
	validator           *validation.ChatRequestValidator
	chatService         *chatService.ChatService
	summaryService      *summaryService.SummaryService
	conversationService *conversationService.ConversationService
}

// NewChatHandlers creates a new ChatHandlers with service layer
func NewChatHandlers(config *app.Config) *ChatHandlers {
	return &ChatHandlers{
		config:              config,
		validator:           validation.NewChatRequestValidator(),
		chatService:         chatService.NewChatService(config.DB, config),
		summaryService:      summaryService.NewSummaryService(config.DB, config),
		conversationService: conversationService.NewConversationService(config.DB),
	}
}

// ChatStreamHandler is the SSE endpoint for streaming chat responses
func (ch *ChatHandlers) ChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(UserContextKey).(string)
	logger.Log.WithField("username", username).Info("Chat stream request received")

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := ch.validator.ValidateChatRequest(req.Message, req.Temperature, req.WarAndPeacePercent, req.ResponseFormat, req.ResponseSchema); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	logger.Log.WithField("message_chars", len(req.Message)).Debug("Processing stream message")

	// Get user from database
	user, err := ch.getUserFromContext(r)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user")
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Check if response writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		ch.sendError(w, http.StatusInternalServerError, "Streaming not supported", nil)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Build service request
	serviceReq := chatService.SendMessageRequest{
		Message:            req.Message,
		ConversationID:     req.ConversationID,
		SystemPrompt:       req.SystemPrompt,
		ResponseFormat:     req.ResponseFormat,
		ResponseSchema:     req.ResponseSchema,
		Model:              req.Model,
		Temperature:        req.Temperature,
		Provider:           req.Provider,
		UseWarAndPeace:     req.UseWarAndPeace,
		WarAndPeacePercent: req.WarAndPeacePercent,
		UserID:             user.ID,
	}

	// Call service to stream message
	chunks, err := ch.chatService.SendMessageStream(serviceReq)
	if err != nil {
		logger.Log.WithError(err).Error("Error from chat service")
		fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", err.Error())
		return
	}

	// Track metadata
	var conversationID, model string
	var temperature *float64
	var usage *llm.ResponseUsage

	// Stream chunks to client using SSE format
	for streamChunk := range chunks {
		if streamChunk.ConvID != "" {
			// First chunk with metadata
			conversationID = streamChunk.ConvID
			model = streamChunk.Model
			temperature = streamChunk.Temperature

			// Send conversation ID
			fmt.Fprintf(w, "data: CONV_ID:%s\n\n", conversationID)
			flusher.Flush()
			logger.Log.WithField("conversation_id", conversationID).Debug("Sent conversation ID to client")

			// Send model
			fmt.Fprintf(w, "data: MODEL:%s\n\n", model)
			flusher.Flush()
			logger.Log.WithField("model", model).Debug("Sent model to client")

			// Send temperature if available
			if temperature != nil {
				fmt.Fprintf(w, "data: TEMPERATURE:%.2f\n\n", *temperature)
				flusher.Flush()
				logger.Log.WithField("temperature", *temperature).Debug("Sent temperature to client")
			}
		} else if streamChunk.Metadata != nil {
			// Final chunk with usage metadata
			usage = streamChunk.Metadata.Usage
			if usage != nil {
				// Build usage JSON with optional fields
				usageJSON := fmt.Sprintf("{\"prompt_tokens\":%d,\"completion_tokens\":%d,\"total_tokens\":%d",
					usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)

				if streamChunk.Metadata.TotalCost != nil {
					usageJSON += fmt.Sprintf(",\"total_cost\":%.6f", *streamChunk.Metadata.TotalCost)
				}
				if streamChunk.Metadata.Latency != nil {
					usageJSON += fmt.Sprintf(",\"latency\":%d", *streamChunk.Metadata.Latency)
				}
				if streamChunk.Metadata.GenerationTime != nil {
					usageJSON += fmt.Sprintf(",\"generation_time\":%d", *streamChunk.Metadata.GenerationTime)
				}
				usageJSON += "}"

				fmt.Fprintf(w, "data: USAGE:%s\n\n", usageJSON)
				flusher.Flush()
				logger.Log.WithField("total_tokens", usage.TotalTokens).Debug("Sent usage data to client")
			}
		} else if streamChunk.Content != "" {
			// Content chunk
			escapedChunk := strings.ReplaceAll(streamChunk.Content, "\n", "\\n")
			fmt.Fprintf(w, "data: %s\n\n", escapedChunk)
			flusher.Flush()
			logger.Log.WithField("chunk_chars", len(streamChunk.Content)).Debug("Sent content chunk")
		}
	}

	// Send completion marker
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// ChatHandler is the REST endpoint for chat (non-streaming)
func (ch *ChatHandlers) ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(UserContextKey).(string)
	logger.Log.WithField("username", username).Info("Chat request received")

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Message == "" {
		ch.sendError(w, http.StatusBadRequest, "Message cannot be empty", nil)
		return
	}

	// Get user from database
	user, err := ch.getUserFromContext(r)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user")
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Build service request
	serviceReq := chatService.SendMessageRequest{
		Message:            req.Message,
		ConversationID:     req.ConversationID,
		SystemPrompt:       req.SystemPrompt,
		ResponseFormat:     req.ResponseFormat,
		ResponseSchema:     req.ResponseSchema,
		Model:              req.Model,
		Temperature:        req.Temperature,
		Provider:           req.Provider,
		UseWarAndPeace:     req.UseWarAndPeace,
		WarAndPeacePercent: req.WarAndPeacePercent,
		UserID:             user.ID,
	}

	// Call service
	response, err := ch.chatService.SendMessage(serviceReq)
	if err != nil {
		logger.Log.WithError(err).Error("Error from chat service")
		ch.sendError(w, http.StatusInternalServerError, "Error processing message", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Response:       response.Response,
		ConversationID: response.ConversationID,
		Model:          response.Model,
	})
}

// GetConversationsHandler returns all conversations for the authenticated user
func (ch *ChatHandlers) GetConversationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(UserContextKey).(string)
	logger.Log.WithField("username", username).Info("Get conversations request")

	// Get user from database
	user, err := ch.config.DB.GetUserByUsername(username)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user")
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Call service
	conversations, err := ch.conversationService.GetUserConversations(user.ID)
	if err != nil {
		logger.Log.WithError(err).Error("Error from conversation service")
		ch.sendError(w, http.StatusInternalServerError, "Error retrieving conversations", err)
		return
	}

	// Convert to response format
	convInfos := make([]ConversationInfo, 0, len(conversations))
	for _, conv := range conversations {
		convInfos = append(convInfos, ConversationInfo{
			ID:                      conv.ID,
			Title:                   conv.Title,
			ResponseFormat:          conv.ResponseFormat,
			ResponseSchema:          conv.ResponseSchema,
			SummarizedUpToMessageID: conv.SummarizedUpToMessageID,
			CreatedAt:               conv.CreatedAt,
			UpdatedAt:               conv.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConversationsResponse{
		Conversations: convInfos,
	})
}

// GetConversationMessagesHandler returns all messages from a specific conversation
func (ch *ChatHandlers) GetConversationMessagesHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(UserContextKey).(string)
	convID := r.PathValue("id")
	logger.Log.WithFields(logrus.Fields{"username": username, "conversation_id": convID}).Info("Get conversation messages request")

	// Get user from database
	user, err := ch.config.DB.GetUserByUsername(username)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user")
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Call service
	messages, err := ch.conversationService.GetConversationMessages(convID, user.ID)
	if err != nil {
		logger.Log.WithError(err).Error("Error from conversation service")
		if strings.Contains(err.Error(), "unauthorized") {
			ch.sendError(w, http.StatusForbidden, "Unauthorized", err)
		} else if strings.Contains(err.Error(), "not found") {
			ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		} else {
			ch.sendError(w, http.StatusInternalServerError, "Error retrieving messages", err)
		}
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
	username := r.Context().Value(UserContextKey).(string)
	convID := r.PathValue("id")
	logger.Log.WithFields(logrus.Fields{"username": username, "conversation_id": convID}).Info("Delete conversation request")

	// Get user from database
	user, err := ch.config.DB.GetUserByUsername(username)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user")
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Call service
	if err := ch.conversationService.DeleteConversation(convID, user.ID); err != nil {
		logger.Log.WithError(err).Error("Error from conversation service")
		if strings.Contains(err.Error(), "unauthorized") {
			ch.sendError(w, http.StatusForbidden, "Unauthorized", err)
		} else if strings.Contains(err.Error(), "not found") {
			ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		} else {
			ch.sendError(w, http.StatusInternalServerError, "Error deleting conversation", err)
		}
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

	models := ch.config.ModelsConfig().GetAvailableModels()

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

	username := r.Context().Value(UserContextKey).(string)
	convID := r.PathValue("id")
	logger.Log.WithFields(logrus.Fields{"username": username, "conversation_id": convID}).Info("Summarize conversation request")

	var req SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is acceptable, use defaults
		req.Model = ""
		req.Temperature = nil
	}

	// Validate request
	if err := ch.validator.ValidateSummarizeRequest(req.Temperature); err != nil {
		ch.sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Get user from database
	user, err := ch.config.DB.GetUserByUsername(username)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user for summarization")
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Build service request
	serviceReq := summaryService.SummarizeRequest{
		ConversationID: convID,
		UserID:         user.ID,
		Model:          req.Model,
		Temperature:    req.Temperature,
	}

	// Call service
	response, err := ch.summaryService.SummarizeConversation(serviceReq)
	if err != nil {
		logger.Log.WithError(err).Error("Error from summary service")
		if strings.Contains(err.Error(), "unauthorized") {
			ch.sendError(w, http.StatusForbidden, "Unauthorized", err)
		} else if strings.Contains(err.Error(), "not found") {
			ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		} else {
			ch.sendError(w, http.StatusInternalServerError, "Error generating summary", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SummarizeResponse{
		Summary:             response.Summary,
		SummarizedUpToMsgID: response.SummarizedUpToMsgID,
		ConversationID:      response.ConversationID,
	})
}

// GetConversationSummariesHandler retrieves all summaries for a conversation
func (ch *ChatHandlers) GetConversationSummariesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ch.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	username := r.Context().Value(UserContextKey).(string)
	convID := r.PathValue("id")
	logger.Log.WithFields(logrus.Fields{"username": username, "conversation_id": convID}).Info("Get summaries request")

	// Get user from database
	user, err := ch.config.DB.GetUserByUsername(username)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user for summaries request")
		ch.sendError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Call service
	summaries, err := ch.summaryService.GetAllSummaries(convID, user.ID)
	if err != nil {
		logger.Log.WithError(err).Error("Error from summary service")
		if strings.Contains(err.Error(), "unauthorized") {
			ch.sendError(w, http.StatusForbidden, "Unauthorized", err)
		} else if strings.Contains(err.Error(), "not found") {
			ch.sendError(w, http.StatusNotFound, "Conversation not found", err)
		} else {
			ch.sendError(w, http.StatusInternalServerError, "Error retrieving summaries", err)
		}
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

// Helper methods

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
	username := r.Context().Value(UserContextKey).(string)
	return ch.config.DB.GetUserByUsername(username)
}
