package main

import (
	"chat-app/internal/api/handlers"
	"chat-app/internal/app"
	"chat-app/internal/config"
	"chat-app/internal/context"
	"chat-app/internal/logger"
	"chat-app/internal/repository/postgres"
	"net/http"
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
	// Load centralized configuration
	logger.Log.Info("Loading application configuration")
	appConfig, err := config.LoadConfig()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to load configuration")
	}
	logger.Log.WithField("model_count", len(appConfig.Models.GetAvailableModels())).Info("Loaded models")

	// Initialize database with config
	logger.Log.Info("Initializing database")
	database, err := postgres.NewPostgresDB(appConfig.Database)
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize database")
	}
	defer database.Close()

	// Load War and Peace text
	logger.Log.Info("Loading War and Peace context")
	warAndPeacePath := "warandpeace.txt"
	if err := context.LoadWarAndPeace(warAndPeacePath); err != nil {
		logger.Log.WithError(err).Warn("Failed to load War and Peace text")
	}

	// Seed demo user
	if err := postgres.SeedDemoUser(database); err != nil {
		logger.Log.WithError(err).Fatal("Failed to seed demo user")
	}

	// Initialize application config with database and centralized config
	appConfiguration := app.NewConfig(database, appConfig)

	// Create chat handlers with dependency injection
	chatHandler := handlers.NewChatHandlers(appConfiguration)

	// Create new ServeMux to use Go 1.22+ routing features for path parameters
	mux := http.NewServeMux()

	// CORS preflight handler for OPTIONS requests
	corsHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
	}

	// Create auth handlers with config and database
	authHandler := handlers.NewAuthHandlers(appConfig, database)

	// Public routes
	mux.HandleFunc("POST /api/login", enableCORS(authHandler.LoginHandler))
	mux.HandleFunc("OPTIONS /api/login", corsHandler)
	mux.HandleFunc("POST /api/register", enableCORS(authHandler.RegisterHandler))
	mux.HandleFunc("OPTIONS /api/register", corsHandler)
	mux.HandleFunc("GET /api/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	mux.HandleFunc("OPTIONS /api/health", corsHandler)
	mux.HandleFunc("GET /api/models", enableCORS(chatHandler.GetModelsHandler))
	mux.HandleFunc("OPTIONS /api/models", corsHandler)

	// Protected routes - use method-based routing (Go 1.22+ native)
	mux.HandleFunc("POST /api/chat", enableCORS(authHandler.AuthMiddleware(chatHandler.ChatHandler)))
	mux.HandleFunc("OPTIONS /api/chat", corsHandler)
	mux.HandleFunc("POST /api/chat/stream", enableCORS(authHandler.AuthMiddleware(chatHandler.ChatStreamHandler)))
	mux.HandleFunc("OPTIONS /api/chat/stream", corsHandler)
	mux.HandleFunc("GET /api/conversations", enableCORS(authHandler.AuthMiddleware(chatHandler.GetConversationsHandler)))
	mux.HandleFunc("OPTIONS /api/conversations", corsHandler)

	// Protected parameterized routes (Go 1.22+ native path parameters with {id})
	mux.HandleFunc("GET /api/conversations/{id}/messages", enableCORS(authHandler.AuthMiddleware(chatHandler.GetConversationMessagesHandler)))
	mux.HandleFunc("OPTIONS /api/conversations/{id}/messages", corsHandler)
	mux.HandleFunc("DELETE /api/conversations/{id}", enableCORS(authHandler.AuthMiddleware(chatHandler.DeleteConversationHandler)))
	mux.HandleFunc("OPTIONS /api/conversations/{id}", corsHandler)
	mux.HandleFunc("POST /api/conversations/{id}/summarize", enableCORS(authHandler.AuthMiddleware(chatHandler.SummarizeConversationHandler)))
	mux.HandleFunc("OPTIONS /api/conversations/{id}/summarize", corsHandler)
	mux.HandleFunc("GET /api/conversations/{id}/summaries", enableCORS(authHandler.AuthMiddleware(chatHandler.GetConversationSummariesHandler)))
	mux.HandleFunc("OPTIONS /api/conversations/{id}/summaries", corsHandler)

	port := appConfig.Server.Port
	logger.Log.WithField("port", port).Info("Server starting")
	logger.Log.WithField("url", "http://localhost:"+port+"/api/health").Info("Health check endpoint")
	logger.Log.WithField("url", "http://localhost:"+port+"/api/login").Info("Login endpoint")
	logger.Log.WithField("url", "http://localhost:"+port+"/api/register").Info("Register endpoint")
	logger.Log.WithField("url", "http://localhost:"+port+"/api/chat").Info("Chat endpoint")
	logger.Log.WithField("url", "http://localhost:"+port+"/api/chat/stream").Info("Chat stream endpoint")
	logger.Log.WithField("url", "http://localhost:"+port+"/api/conversations").Info("Conversations endpoint")
	logger.Log.WithField("url", "http://localhost:"+port+"/api/conversations/{id}/messages").Info("Conversation messages endpoint")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Log.WithError(err).Fatal("Server failed to start")
	}
}
