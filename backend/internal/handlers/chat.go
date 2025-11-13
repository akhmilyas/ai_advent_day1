package handlers

import (
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

type ChatHandlers struct{}

func NewChatHandlers() *ChatHandlers {
	return &ChatHandlers{}
}

// ChatHandler is the REST endpoint for chat
func (ch *ChatHandlers) ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	log.Printf("Chat request from user: %s", username)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	log.Printf("[CHAT] User input: %s", req.Message)

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[CHAT] Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get or create conversation
	var conversation *db.Conversation
	if req.ConversationID != "" {
		conversation, err = db.GetConversation(req.ConversationID)
		if err != nil {
			log.Printf("[CHAT] Error getting conversation: %v", err)
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}
		// Verify user owns this conversation
		if conversation.UserID != user.ID {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}
	} else {
		// Create new conversation with first message as title and specified format
		title := req.Message
		// Use rune slicing to avoid cutting multi-byte UTF-8 characters
		runes := []rune(title)
		if len(runes) > 100 {
			title = string(runes[:100])
		}
		conversation, err = db.CreateConversation(user.ID, title, req.ResponseFormat, req.ResponseSchema)
		if err != nil {
			log.Printf("[CHAT] Error creating conversation: %v", err)
			http.Error(w, "Error creating conversation", http.StatusInternalServerError)
			return
		}
	}

	// Validate model if provided
	model := req.Model
	if model != "" && !config.IsValidModel(model) {
		http.Error(w, "Invalid model specified", http.StatusBadRequest)
		return
	}

	// Add user message to database (user messages don't have a model, temperature, provider, or usage data)
	if _, err := db.AddMessage(conversation.ID, "user", req.Message, "", nil, "", "", nil, nil, nil, nil, nil, nil); err != nil {
		log.Printf("[CHAT] Error adding user message: %v", err)
		http.Error(w, "Error saving message", http.StatusInternalServerError)
		return
	}

	// Get conversation history
	currentHistory, err := db.GetConversationMessages(conversation.ID)
	if err != nil {
		log.Printf("[CHAT] Error getting conversation history: %v", err)
		http.Error(w, "Error retrieving conversation history", http.StatusInternalServerError)
		return
	}

	log.Printf("[CHAT] Conversation history length: %d messages", len(currentHistory))

	// Get LLM provider based on request
	provider := llm.GetProviderFromString(req.Provider)
	log.Printf("[CHAT] Using provider: %T", provider)

	// Get response with full conversation history
	response, err := provider.ChatWithHistory(currentHistory, req.SystemPrompt, conversation.ResponseFormat, model, req.Temperature)
	if err != nil {
		log.Printf("[CHAT] Error from LLM: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: err.Error(),
		})
		return
	}

	log.Printf("[CHAT] LLM response: %s", response)

	// Determine which model was actually used
	usedModel := model
	if usedModel == "" {
		usedModel = provider.GetDefaultModel()
	}

	// Add assistant response to database with model, temperature, and provider (no usage data for non-streaming)
	if _, err := db.AddMessage(conversation.ID, "assistant", response, usedModel, req.Temperature, req.Provider, "", nil, nil, nil, nil, nil, nil); err != nil {
		log.Printf("[CHAT] Error adding assistant message: %v", err)
		http.Error(w, "Error saving response", http.StatusInternalServerError)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	log.Printf("Chat stream request from user: %s", username)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	log.Printf("[CHAT] User input (stream): %s", req.Message)

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[CHAT] Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get or create conversation
	var conversation *db.Conversation
	if req.ConversationID != "" {
		conversation, err = db.GetConversation(req.ConversationID)
		if err != nil {
			log.Printf("[CHAT] Error getting conversation: %v", err)
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}
		// Verify user owns this conversation
		if conversation.UserID != user.ID {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}
	} else {
		// Create new conversation with first message as title and specified format
		title := req.Message
		// Use rune slicing to avoid cutting multi-byte UTF-8 characters
		runes := []rune(title)
		if len(runes) > 100 {
			title = string(runes[:100])
		}
		conversation, err = db.CreateConversation(user.ID, title, req.ResponseFormat, req.ResponseSchema)
		if err != nil {
			log.Printf("[CHAT] Error creating conversation: %v", err)
			http.Error(w, "Error creating conversation", http.StatusInternalServerError)
			return
		}
	}

	// Validate model if provided
	model := req.Model
	if model != "" && !config.IsValidModel(model) {
		http.Error(w, "Invalid model specified", http.StatusBadRequest)
		return
	}

	// Add user message to database (user messages don't have a model, temperature, provider, or usage data)
	if _, err := db.AddMessage(conversation.ID, "user", req.Message, "", nil, "", "", nil, nil, nil, nil, nil, nil); err != nil {
		log.Printf("[CHAT] Error adding user message: %v", err)
		http.Error(w, "Error saving message", http.StatusInternalServerError)
		return
	}

	// Check if there's an active summary for this conversation
	activeSummary, err := db.GetActiveSummary(conversation.ID)
	var currentHistory []llm.Message

	if err == nil && activeSummary != nil {
		// Active summary exists - use it instead of full history
		log.Printf("[CHAT] Using active summary (usage count: %d)", activeSummary.UsageCount)

		// Get messages after the summarized point
		if activeSummary.SummarizedUpToMessageID != nil {
			newMessages, err := db.GetMessagesAfterMessage(conversation.ID, *activeSummary.SummarizedUpToMessageID)
			if err != nil {
				log.Printf("[CHAT] Error getting messages after summary: %v", err)
				http.Error(w, "Error retrieving conversation history", http.StatusInternalServerError)
				return
			}
			currentHistory = newMessages
			log.Printf("[CHAT] Using summary + %d new messages", len(newMessages))
		} else {
			// No messages after summary (shouldn't happen, but handle gracefully)
			currentHistory = []llm.Message{}
			log.Printf("[CHAT] Using summary with no new messages")
		}

		// Increment summary usage count
		if err := db.IncrementSummaryUsageCount(activeSummary.ID); err != nil {
			log.Printf("[CHAT] Warning: failed to increment summary usage count: %v", err)
		}
	} else {
		// No active summary - use full conversation history
		currentHistory, err = db.GetConversationMessages(conversation.ID)
		if err != nil {
			log.Printf("[CHAT] Error getting conversation history: %v", err)
			http.Error(w, "Error retrieving conversation history", http.StatusInternalServerError)
			return
		}
		log.Printf("[CHAT] Using full conversation history: %d messages", len(currentHistory))
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Build the system prompt based on conversation's response format (stored in DB)
	// If there's an active summary, combine it with the user's custom prompt
	var effectiveSystemPrompt string
	if activeSummary != nil {
		// Summary exists - use it as context and add user's system prompt
		summaryContext := fmt.Sprintf("Previous conversation summary:\n%s\n\n", activeSummary.SummaryContent)

		if conversation.ResponseFormat == "json" && conversation.ResponseSchema != "" {
			effectiveSystemPrompt = summaryContext + fmt.Sprintf("You must respond ONLY with valid JSON that matches this exact schema. Do not include any explanatory text, markdown formatting, or code blocks - just the raw JSON.\n\nSchema:\n%s\n\nRemember: Your entire response must be valid JSON matching this schema.", conversation.ResponseSchema)
		} else if conversation.ResponseFormat == "xml" && conversation.ResponseSchema != "" {
			effectiveSystemPrompt = summaryContext + fmt.Sprintf("You must respond ONLY with valid XML that matches this exact schema. Do not include any explanatory text, markdown formatting, or code blocks - just the raw XML.\n\nSchema:\n%s\n\nRemember: Your entire response must be valid XML matching this schema.", conversation.ResponseSchema)
		} else {
			// For text format, combine summary with user's custom system prompt
			effectiveSystemPrompt = summaryContext + req.SystemPrompt
		}
		log.Printf("[CHAT] Using summary as context with user prompt")
	} else if conversation.ResponseFormat == "json" && conversation.ResponseSchema != "" {
		effectiveSystemPrompt = fmt.Sprintf("You must respond ONLY with valid JSON that matches this exact schema. Do not include any explanatory text, markdown formatting, or code blocks - just the raw JSON.\n\nSchema:\n%s\n\nRemember: Your entire response must be valid JSON matching this schema.", conversation.ResponseSchema)
	} else if conversation.ResponseFormat == "xml" && conversation.ResponseSchema != "" {
		effectiveSystemPrompt = fmt.Sprintf("You must respond ONLY with valid XML that matches this exact schema. Do not include any explanatory text, markdown formatting, or code blocks - just the raw XML.\n\nSchema:\n%s\n\nRemember: Your entire response must be valid XML matching this schema.", conversation.ResponseSchema)
	} else {
		// For text format, use custom system prompt from request
		effectiveSystemPrompt = req.SystemPrompt
	}

	// Append War and Peace context if requested
	if req.UseWarAndPeace {
		warAndPeaceText := context.GetWarAndPeace()
		if warAndPeaceText != "" {
			// Calculate how much of the text to include based on percentage
			percent := req.WarAndPeacePercent
			if percent <= 0 || percent > 100 {
				percent = 100 // Default to 100% if invalid
			}

			// Calculate the number of characters to include
			totalChars := len(warAndPeaceText)
			charsToInclude := (totalChars * percent) / 100

			// Get the substring from the beginning
			textToAppend := warAndPeaceText[:charsToInclude]

			effectiveSystemPrompt = effectiveSystemPrompt + "\n\nContext (War and Peace by Leo Tolstoy):\n" + textToAppend
			log.Printf("[CHAT] Appended War and Peace context: %d%% (%.2f MB of %.2f MB)",
				percent,
				float64(len(textToAppend))/1024/1024,
				float64(totalChars)/1024/1024)
		} else {
			log.Printf("[CHAT] Warning: War and Peace text not loaded")
		}
	}

	log.Printf("[CHAT] Using conversation format: %s", conversation.ResponseFormat)

	// Get LLM provider based on request
	provider := llm.GetProviderFromString(req.Provider)
	log.Printf("[CHAT] Using provider for streaming: %T", provider)

	// Get streaming response from LLM
	chunks, err := provider.ChatWithHistoryStream(currentHistory, effectiveSystemPrompt, conversation.ResponseFormat, model, req.Temperature)
	if err != nil {
		log.Printf("[CHAT] Error from LLM stream: %v", err)
		fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", err.Error())
		return
	}

	// Determine which model was actually used
	usedModel := model
	if usedModel == "" {
		usedModel = provider.GetDefaultModel()
	}

	// Send conversation ID as first event
	fmt.Fprintf(w, "data: CONV_ID:%s\n\n", conversation.ID)
	flusher.Flush()
	log.Printf("[CHAT] Sent conversation ID: %s", conversation.ID)

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
		if _, err := db.AddMessage(conversation.ID, "assistant", fullResponse, usedModel, req.Temperature, req.Provider,
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	log.Printf("Get conversations request from user: %s", username)

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[CHAT] Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get all conversations for user
	conversations, err := db.GetConversationsByUser(user.ID)
	if err != nil {
		log.Printf("[CHAT] Error getting conversations: %v", err)
		http.Error(w, "Error retrieving conversations", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	convInfos := make([]ConversationInfo, 0, len(conversations))
	for _, conv := range conversations {
		// Get active summary for this conversation if it exists
		var summarizedUpToMsgID *string
		if summary, err := db.GetActiveSummary(conv.ID); err == nil && summary != nil {
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
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get conversation and verify ownership
	conversation, err := db.GetConversation(convID)
	if err != nil {
		log.Printf("[CHAT] Error getting conversation: %v", err)
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// Verify user owns this conversation
	if conversation.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get messages for conversation
	messages, err := db.GetConversationMessagesWithDetails(convID)
	if err != nil {
		log.Printf("[CHAT] Error getting messages: %v", err)
		http.Error(w, "Error retrieving messages", http.StatusInternalServerError)
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
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get conversation and verify ownership
	conversation, err := db.GetConversation(convID)
	if err != nil {
		log.Printf("[CHAT] Error getting conversation: %v", err)
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// Verify user owns this conversation
	if conversation.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Delete the conversation
	if err := db.DeleteConversation(convID); err != nil {
		log.Printf("[CHAT] Error deleting conversation: %v", err)
		http.Error(w, "Error deleting conversation", http.StatusInternalServerError)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models := config.GetAvailableModels()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ModelsResponse{
		Models: models,
	})
}

// SummarizeConversationHandler creates a summary of the conversation
func (ch *ChatHandlers) SummarizeConversationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get conversation and verify ownership
	conversation, err := db.GetConversation(convID)
	if err != nil {
		log.Printf("[SUMMARIZE] Error getting conversation: %v", err)
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// Verify user owns this conversation
	if conversation.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Check if there's an existing active summary
	activeSummary, err := db.GetActiveSummary(convID)
	var messagesToSummarize []llm.Message
	var lastMessageID *string

	if err != nil || activeSummary == nil {
		// No active summary exists - summarize all messages
		log.Printf("[SUMMARIZE] No active summary found, summarizing all messages")
		messagesToSummarize, err = db.GetConversationMessages(convID)
		if err != nil {
			log.Printf("[SUMMARIZE] Error getting conversation messages: %v", err)
			http.Error(w, "Error retrieving messages", http.StatusInternalServerError)
			return
		}

		// Get the last message ID
		lastMessageID, err = db.GetLastMessageID(convID)
		if err != nil {
			log.Printf("[SUMMARIZE] Error getting last message ID: %v", err)
			http.Error(w, "Error retrieving last message", http.StatusInternalServerError)
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
			newMessages, err := db.GetMessagesAfterMessage(convID, *activeSummary.SummarizedUpToMessageID)
			if err != nil {
				log.Printf("[SUMMARIZE] Error getting messages after last summarized: %v", err)
				http.Error(w, "Error retrieving new messages", http.StatusInternalServerError)
				return
			}
			messagesToSummarize = append(messagesToSummarize, newMessages...)
		}

		// Get the last message ID
		lastMessageID, err = db.GetLastMessageID(convID)
		if err != nil {
			log.Printf("[SUMMARIZE] Error getting last message ID: %v", err)
			http.Error(w, "Error retrieving last message", http.StatusInternalServerError)
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
	model := req.Model
	if model != "" && !config.IsValidModel(model) {
		http.Error(w, "Invalid model specified", http.StatusBadRequest)
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
	summaryContent, err := provider.ChatForSummarization(messagesToSummarize, summarizationPrompt, model, req.Temperature)
	if err != nil {
		log.Printf("[SUMMARIZE] Error from LLM: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SummarizeResponse{
			Error: err.Error(),
		})
		return
	}

	log.Printf("[SUMMARIZE] Generated summary: %s", summaryContent)

	// Create new summary in database
	summary, err := db.CreateSummary(convID, summaryContent, lastMessageID)
	if err != nil {
		log.Printf("[SUMMARIZE] Error creating summary: %v", err)
		http.Error(w, "Error saving summary", http.StatusInternalServerError)
		return
	}

	// Update conversation to use this new summary
	if err := db.UpdateConversationActiveSummary(convID, summary.ID); err != nil {
		log.Printf("[SUMMARIZE] Error updating active summary: %v", err)
		http.Error(w, "Error updating conversation", http.StatusInternalServerError)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Context().Value(auth.UserContextKey).(string)
	convID := r.PathValue("id")
	log.Printf("Get summaries request from user: %s for conversation: %s", username, convID)

	// Get user from database
	user, err := db.GetUserByUsername(username)
	if err != nil {
		log.Printf("[SUMMARIES] Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get conversation and verify ownership
	conversation, err := db.GetConversation(convID)
	if err != nil {
		log.Printf("[SUMMARIES] Error getting conversation: %v", err)
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// Verify user owns this conversation
	if conversation.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get all summaries for conversation
	summaries, err := db.GetAllSummaries(convID)
	if err != nil {
		log.Printf("[SUMMARIES] Error getting summaries: %v", err)
		http.Error(w, "Error retrieving summaries", http.StatusInternalServerError)
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
