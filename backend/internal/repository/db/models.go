package db

import "time"

// User represents a user in the database
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    string
}

// Conversation represents a conversation in the database
type Conversation struct {
	ID              string
	UserID          string
	Title           string
	ResponseFormat  string
	ResponseSchema  string
	ActiveSummaryID *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ConversationSummary represents a summary of conversation messages
type ConversationSummary struct {
	ID                      string
	ConversationID          string
	SummaryContent          string
	SummarizedUpToMessageID *string
	UsageCount              int
	CreatedAt               time.Time
}

// Message represents a message in a conversation
type Message struct {
	ID               string
	ConversationID   string
	Role             string
	Content          string
	Model            string
	Temperature      *float64
	Provider         string   // LLM provider used (openrouter, genkit)
	GenerationID     string
	PromptTokens     *int
	CompletionTokens *int
	TotalTokens      *int
	TotalCost        *float64
	Latency          *int // Time to first token in milliseconds
	GenerationTime   *int // Total generation time in milliseconds
	CreatedAt        time.Time
}
