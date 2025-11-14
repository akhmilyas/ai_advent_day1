package db

import (
	"chat-app/internal/llm"
)

// Database defines the interface for all database operations
// This allows for easier testing through mocking and decouples the handlers from the specific database implementation
type Database interface {
	// Users
	GetUserByUsername(username string) (*User, error)
	CreateUser(username, email, passwordHash string) (*User, error)

	// Conversations
	GetConversation(id string) (*Conversation, error)
	CreateConversation(userID, title, format, schema string) (*Conversation, error)
	GetConversationsByUser(userID string) ([]Conversation, error)
	DeleteConversation(id string) error

	// Messages
	AddMessage(conversationID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*Message, error)
	GetConversationMessages(conversationID string) ([]llm.Message, error)
	GetConversationMessagesWithDetails(conversationID string) ([]Message, error)
	GetMessagesAfterMessage(conversationID, afterMessageID string) ([]llm.Message, error)
	GetLastMessageID(conversationID string) (*string, error)

	// Summaries
	GetActiveSummary(conversationID string) (*ConversationSummary, error)
	CreateSummary(conversationID, summaryContent string, summarizedUpToMessageID *string) (*ConversationSummary, error)
	GetAllSummaries(conversationID string) ([]ConversationSummary, error)
	IncrementSummaryUsageCount(summaryID string) error
	UpdateConversationActiveSummary(conversationID, summaryID string) error
}
