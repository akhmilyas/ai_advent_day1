package db

import (
	"chat-app/internal/llm"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// Conversation represents a conversation in the database
type Conversation struct {
	ID             string
	UserID         string
	Title          string
	ResponseFormat string
	ResponseSchema string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Message represents a message in a conversation
type Message struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	CreatedAt      time.Time
}

// CreateConversation creates a new conversation for a user
func CreateConversation(userID string, title string, responseFormat string, responseSchema string) (*Conversation, error) {
	db := GetDB()

	convID := uuid.New().String()
	var createdAt, updatedAt time.Time

	// Default to 'text' format if not specified
	if responseFormat == "" {
		responseFormat = "text"
	}

	query := `
	INSERT INTO conversations (id, user_id, title, response_format, response_schema)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at, updated_at
	`

	err := db.QueryRow(query, convID, userID, title, responseFormat, responseSchema).Scan(&convID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("error creating conversation: %w", err)
	}

	log.Printf("[DB] Created new conversation: %s for user: %s with format: %s", convID, userID, responseFormat)

	return &Conversation{
		ID:             convID,
		UserID:         userID,
		Title:          title,
		ResponseFormat: responseFormat,
		ResponseSchema: responseSchema,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

// GetConversationsByUser retrieves all conversations for a user
func GetConversationsByUser(userID string) ([]Conversation, error) {
	db := GetDB()

	query := `
	SELECT id, user_id, title, COALESCE(response_format, 'text'), COALESCE(response_schema, ''), created_at, updated_at
	FROM conversations
	WHERE user_id = $1
	ORDER BY updated_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying conversations: %w", err)
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		if err := rows.Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.ResponseFormat, &conv.ResponseSchema, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// GetConversation retrieves a specific conversation
func GetConversation(convID string) (*Conversation, error) {
	db := GetDB()

	var conv Conversation
	query := `
	SELECT id, user_id, title, COALESCE(response_format, 'text'), COALESCE(response_schema, ''), created_at, updated_at
	FROM conversations
	WHERE id = $1
	`

	err := db.QueryRow(query, convID).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.ResponseFormat, &conv.ResponseSchema, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error retrieving conversation: %w", err)
	}

	return &conv, nil
}

// AddMessage adds a message to a conversation
func AddMessage(conversationID string, role, content string) (*Message, error) {
	db := GetDB()

	msgID := uuid.New().String()
	var createdAt time.Time

	query := `
	INSERT INTO messages (id, conversation_id, role, content)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at
	`

	err := db.QueryRow(query, msgID, conversationID, role, content).Scan(&msgID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("error adding message: %w", err)
	}

	// Update conversation updated_at timestamp
	updateQuery := `UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	if _, err := db.Exec(updateQuery, conversationID); err != nil {
		log.Printf("[DB] Warning: error updating conversation timestamp: %v", err)
	}

	log.Printf("[DB] Added message to conversation %s", conversationID)

	return &Message{
		ID:             msgID,
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		CreatedAt:      createdAt,
	}, nil
}

// GetConversationMessages retrieves all messages from a conversation in LLM format
func GetConversationMessages(conversationID string) ([]llm.Message, error) {
	db := GetDB()

	query := `
	SELECT role, content
	FROM messages
	WHERE conversation_id = $1
	ORDER BY created_at ASC
	`

	rows, err := db.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("error querying messages: %w", err)
	}
	defer rows.Close()

	var messages []llm.Message
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, fmt.Errorf("error scanning message: %w", err)
		}
		messages = append(messages, llm.Message{
			Role:    role,
			Content: content,
		})
	}

	return messages, nil
}

// GetConversationMessagesWithDetails retrieves all messages with full details for frontend display
func GetConversationMessagesWithDetails(conversationID string) ([]Message, error) {
	db := GetDB()

	query := `
	SELECT id, conversation_id, role, content, created_at
	FROM messages
	WHERE conversation_id = $1
	ORDER BY created_at ASC
	`

	rows, err := db.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("error querying messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// DeleteConversation deletes a conversation and all its messages
func DeleteConversation(convID string) error {
	db := GetDB()

	query := `DELETE FROM conversations WHERE id = $1`
	_, err := db.Exec(query, convID)
	if err != nil {
		return fmt.Errorf("error deleting conversation: %w", err)
	}

	log.Printf("[DB] Deleted conversation: %s", convID)
	return nil
}

