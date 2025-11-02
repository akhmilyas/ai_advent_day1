package handlers

import (
	"chat-app/internal/auth"
	"chat-app/internal/llm"
	"encoding/json"
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type ChatRequest struct {
	Message  string          `json:"message,omitempty"`
	Messages []llm.Message   `json:"messages,omitempty"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

type WSMessage struct {
	Type    string `json:"type"` // "start", "chunk", "end", "error"
	Content string `json:"content"`
}

// REST endpoint for non-streaming chat
func ChatHandler(w http.ResponseWriter, r *http.Request) {
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

	// Support both single message and conversation history
	var response string
	var err error

	if len(req.Messages) > 0 {
		// Use conversation history if provided
		log.Printf("[CHAT] User input with %d messages in history", len(req.Messages))
		log.Printf("[CHAT] Last message: %s", req.Messages[len(req.Messages)-1].Content)
		response, err = llm.ChatWithHistory(req.Messages)
	} else if req.Message != "" {
		// Fall back to single message mode for backward compatibility
		log.Printf("[CHAT] User input: %s", req.Message)
		response, err = llm.Chat(req.Message)
	} else {
		http.Error(w, "Message or messages array cannot be empty", http.StatusBadRequest)
		return
	}

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Response: response,
	})
}

// WebSocket endpoint for streaming chat
func ChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(auth.UserContextKey).(string)
	log.Printf("WebSocket connection from user: %s", username)

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("Failed to accept websocket: %v", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "")

	ctx := r.Context()

	// Maintain conversation history for this connection
	var conversationHistory []llm.Message

	for {
		var req ChatRequest
		err := wsjson.Read(ctx, conn, &req)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		if req.Message == "" {
			continue
		}

		log.Printf("[STREAM] User input: %s", req.Message)
		log.Printf("[STREAM] Conversation history length: %d messages", len(conversationHistory))

		// Add user message to conversation history
		conversationHistory = append(conversationHistory, llm.Message{
			Role:    "user",
			Content: req.Message,
		})

		// Send start message
		err = wsjson.Write(ctx, conn, WSMessage{
			Type:    "start",
			Content: "",
		})
		if err != nil {
			log.Printf("Error sending start message: %v", err)
			break
		}

		// Collect streamed response for logging
		var fullResponse string

		// Stream the response using conversation history
		streamErr := llm.StreamChatWithHistory(conversationHistory, func(chunk string) error {
			fullResponse += chunk
			return wsjson.Write(ctx, conn, WSMessage{
				Type:    "chunk",
				Content: chunk,
			})
		})

		if streamErr != nil {
			log.Printf("[STREAM] Error streaming: %v", streamErr)
			wsjson.Write(ctx, conn, WSMessage{
				Type:    "error",
				Content: streamErr.Error(),
			})
			continue
		}

		log.Printf("[STREAM] LLM response: %s", fullResponse)

		// Add assistant response to conversation history
		conversationHistory = append(conversationHistory, llm.Message{
			Role:    "assistant",
			Content: fullResponse,
		})

		// Send end message
		err = wsjson.Write(ctx, conn, WSMessage{
			Type:    "end",
			Content: "",
		})
		if err != nil {
			log.Printf("Error sending end message: %v", err)
			break
		}
	}

	conn.Close(websocket.StatusNormalClosure, "")
}
