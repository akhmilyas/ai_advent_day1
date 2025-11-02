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
	Message string `json:"message"`
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

	if req.Message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	response, err := llm.Chat(req.Message)
	if err != nil {
		log.Printf("Error from LLM: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: err.Error(),
		})
		return
	}

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

		log.Printf("Streaming chat request: %s", req.Message)

		// Send start message
		err = wsjson.Write(ctx, conn, WSMessage{
			Type:    "start",
			Content: "",
		})
		if err != nil {
			log.Printf("Error sending start message: %v", err)
			break
		}

		// Stream the response
		streamErr := llm.StreamChat(req.Message, func(chunk string) error {
			return wsjson.Write(ctx, conn, WSMessage{
				Type:    "chunk",
				Content: chunk,
			})
		})

		if streamErr != nil {
			log.Printf("Error streaming: %v", streamErr)
			wsjson.Write(ctx, conn, WSMessage{
				Type:    "error",
				Content: streamErr.Error(),
			})
			continue
		}

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
