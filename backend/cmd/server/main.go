package main

import (
	"chat-app/internal/auth"
	"chat-app/internal/db"
	"chat-app/internal/handlers"
	"log"
	"net/http"
	"os"
)


func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}


func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database
	log.Printf("Initializing database...")
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()

	// Seed demo user
	if err := db.SeedDemoUser(); err != nil {
		log.Fatalf("Failed to seed demo user: %v", err)
	}

	// Create chat handlers
	chatHandler := handlers.NewChatHandlers()

	// Create new ServeMux to use Go 1.22+ routing features for path parameters
	mux := http.NewServeMux()

	// CORS preflight handler for OPTIONS requests
	corsHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
	}

	// Public routes
	mux.HandleFunc("POST /api/login", enableCORS(auth.LoginHandler))
	mux.HandleFunc("OPTIONS /api/login", corsHandler)
	mux.HandleFunc("POST /api/register", enableCORS(auth.RegisterHandler))
	mux.HandleFunc("OPTIONS /api/register", corsHandler)
	mux.HandleFunc("GET /api/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	mux.HandleFunc("OPTIONS /api/health", corsHandler)

	// Protected routes - use method-based routing (Go 1.22+ native)
	mux.HandleFunc("POST /api/chat", enableCORS(auth.AuthMiddleware(chatHandler.ChatHandler)))
	mux.HandleFunc("OPTIONS /api/chat", corsHandler)
	mux.HandleFunc("POST /api/chat/stream", enableCORS(auth.AuthMiddleware(chatHandler.ChatStreamHandler)))
	mux.HandleFunc("OPTIONS /api/chat/stream", corsHandler)
	mux.HandleFunc("GET /api/conversations", enableCORS(auth.AuthMiddleware(chatHandler.GetConversationsHandler)))
	mux.HandleFunc("OPTIONS /api/conversations", corsHandler)

	// Protected parameterized routes (Go 1.22+ native path parameters with {id})
	mux.HandleFunc("GET /api/conversations/{id}/messages", enableCORS(auth.AuthMiddleware(chatHandler.GetConversationMessagesHandler)))
	mux.HandleFunc("OPTIONS /api/conversations/{id}/messages", corsHandler)
	mux.HandleFunc("DELETE /api/conversations/{id}", enableCORS(auth.AuthMiddleware(chatHandler.DeleteConversationHandler)))
	mux.HandleFunc("OPTIONS /api/conversations/{id}", corsHandler)

	log.Printf("Server starting on port %s", port)
	log.Printf("Health check: http://localhost:%s/api/health", port)
	log.Printf("Login endpoint: http://localhost:%s/api/login", port)
	log.Printf("Register endpoint: http://localhost:%s/api/register", port)
	log.Printf("Chat endpoint: http://localhost:%s/api/chat", port)
	log.Printf("Chat stream endpoint: http://localhost:%s/api/chat/stream", port)
	log.Printf("Conversations endpoint: http://localhost:%s/api/conversations", port)
	log.Printf("Conversation messages endpoint: http://localhost:%s/api/conversations/{id}/messages", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
