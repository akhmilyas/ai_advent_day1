package handlers

import (
	"chat-app/internal/auth"
	"chat-app/internal/db"
	"chat-app/internal/llm"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type ChatRequest struct {
	Message         string        `json:"message,omitempty"`
	Messages        []llm.Message `json:"messages,omitempty"`
	ConversationID  int           `json:"conversation_id,omitempty"`
	SystemPrompt    string        `json:"system_prompt,omitempty"`
}

type ChatResponse struct {
	Response       string `json:"response"`
	ConversationID int    `json:"conversation_id,omitempty"`
	Model          string `json:"model,omitempty"`
	Error          string `json:"error,omitempty"`
}

type ConversationInfo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ConversationsResponse struct {
	Conversations []ConversationInfo `json:"conversations"`
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
	if req.ConversationID > 0 {
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
		// Create new conversation with first message as title
		title := req.Message
		if len(title) > 100 {
			title = title[:100]
		}
		conversation, err = db.CreateConversation(user.ID, title)
		if err != nil {
			log.Printf("[CHAT] Error creating conversation: %v", err)
			http.Error(w, "Error creating conversation", http.StatusInternalServerError)
			return
		}
	}

	// Add user message to database
	if _, err := db.AddMessage(conversation.ID, "user", req.Message); err != nil {
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

	// Get response with full conversation history
	response, err := llm.ChatWithHistory(currentHistory, req.SystemPrompt)
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

	// Add assistant response to database
	if _, err := db.AddMessage(conversation.ID, "assistant", response); err != nil {
		log.Printf("[CHAT] Error adding assistant message: %v", err)
		http.Error(w, "Error saving response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Response:       response,
		ConversationID: conversation.ID,
		Model:          llm.GetModel(),
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
	if req.ConversationID > 0 {
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
		// Create new conversation with first message as title
		title := req.Message
		if len(title) > 100 {
			title = title[:100]
		}
		conversation, err = db.CreateConversation(user.ID, title)
		if err != nil {
			log.Printf("[CHAT] Error creating conversation: %v", err)
			http.Error(w, "Error creating conversation", http.StatusInternalServerError)
			return
		}
	}

	// Add user message to database
	if _, err := db.AddMessage(conversation.ID, "user", req.Message); err != nil {
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

	// Get streaming response from LLM
	chunks, err := llm.ChatWithHistoryStream(currentHistory, req.SystemPrompt)
	if err != nil {
		log.Printf("[CHAT] Error from LLM stream: %v", err)
		fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", err.Error())
		return
	}

	// Send conversation ID as first event
	fmt.Fprintf(w, "data: CONV_ID:%d\n\n", conversation.ID)
	flusher.Flush()
	log.Printf("[CHAT] Sent conversation ID: %d", conversation.ID)

	// Send model as second event
	model := llm.GetModel()
	fmt.Fprintf(w, "data: MODEL:%s\n\n", model)
	flusher.Flush()
	log.Printf("[CHAT] Sent model: %s", model)

	// Buffer to accumulate the full response
	var fullResponse string

	// Stream chunks to client using SSE format
	for chunk := range chunks {
		fullResponse += chunk
		// Escape newlines in chunk for SSE format
		escapedChunk := strings.ReplaceAll(chunk, "\n", "\\n")
		// Send chunk as SSE event
		fmt.Fprintf(w, "data: %s\n\n", escapedChunk)
		flusher.Flush()
		log.Printf("[CHAT] Sent chunk: %q", chunk)
	}

	// Add assistant response to database after streaming completes
	if fullResponse != "" {
		if _, err := db.AddMessage(conversation.ID, "assistant", fullResponse); err != nil {
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
		convInfos = append(convInfos, ConversationInfo{
			ID:        conv.ID,
			Title:     conv.Title,
			CreatedAt: conv.CreatedAt.String(),
			UpdatedAt: conv.UpdatedAt.String(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConversationsResponse{
		Conversations: convInfos,
	})
}
