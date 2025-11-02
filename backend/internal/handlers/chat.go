package handlers

import (
	"chat-app/internal/auth"
	"chat-app/internal/conversation"
	"chat-app/internal/llm"
	"encoding/json"
	"log"
	"net/http"
)

type ChatRequest struct {
	Message  string          `json:"message,omitempty"`
	Messages []llm.Message   `json:"messages,omitempty"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

type ChatHandlers struct {
	sessionManager *conversation.SessionManager
}

func NewChatHandlers(sm *conversation.SessionManager) *ChatHandlers {
	return &ChatHandlers{
		sessionManager: sm,
	}
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

	// Create a simple session ID based on username for per-user history
	sessionID := username
	session := ch.sessionManager.GetOrCreateSession(sessionID, username)

	// Add user message to conversation history
	session.AddUserMessage(req.Message)
	currentHistory := session.GetMessages()
	log.Printf("[CHAT] Conversation history length: %d messages", len(currentHistory))

	// Get response with full conversation history
	response, err := llm.ChatWithHistory(currentHistory)
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

	// Add assistant response to conversation history
	session.AddAssistantMessage(response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Response: response,
	})
}

