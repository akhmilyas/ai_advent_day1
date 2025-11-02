package db

import (
	"chat-app/internal/llm"
	"fmt"
	"log"
	"time"
)

// Conversation represents a conversation in the database
type Conversation struct {
	ID        int
	UserID    int
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Message represents a message in a conversation
type Message struct {
	ID             int
	ConversationID int
	Role           string
	Content        string
	CreatedAt      time.Time
}

// CreateConversation creates a new conversation for a user
func CreateConversation(userID int, title string) (*Conversation, error) {
	db := GetDB()

	var convID int
	var createdAt, updatedAt time.Time

	query := `
	INSERT INTO conversations (user_id, title)
	VALUES ($1, $2)
	RETURNING id, created_at, updated_at
	`

	err := db.QueryRow(query, userID, title).Scan(&convID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("error creating conversation: %w", err)
	}

	log.Printf("[DB] Created new conversation: %d for user: %d", convID, userID)

	return &Conversation{
		ID:        convID,
		UserID:    userID,
		Title:     title,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// GetConversationsByUser retrieves all conversations for a user
func GetConversationsByUser(userID int) ([]Conversation, error) {
	db := GetDB()

	query := `
	SELECT id, user_id, title, created_at, updated_at
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
		if err := rows.Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// GetConversation retrieves a specific conversation
func GetConversation(convID int) (*Conversation, error) {
	db := GetDB()

	var conv Conversation
	query := `
	SELECT id, user_id, title, created_at, updated_at
	FROM conversations
	WHERE id = $1
	`

	err := db.QueryRow(query, convID).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error retrieving conversation: %w", err)
	}

	return &conv, nil
}

// AddMessage adds a message to a conversation
func AddMessage(conversationID int, role, content string) (*Message, error) {
	db := GetDB()

	var msgID int
	var createdAt time.Time

	query := `
	INSERT INTO messages (conversation_id, role, content)
	VALUES ($1, $2, $3)
	RETURNING id, created_at
	`

	err := db.QueryRow(query, conversationID, role, content).Scan(&msgID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("error adding message: %w", err)
	}

	// Update conversation updated_at timestamp
	updateQuery := `UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	if _, err := db.Exec(updateQuery, conversationID); err != nil {
		log.Printf("[DB] Warning: error updating conversation timestamp: %v", err)
	}

	log.Printf("[DB] Added message to conversation %d", conversationID)

	return &Message{
		ID:             msgID,
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		CreatedAt:      createdAt,
	}, nil
}

// GetConversationMessages retrieves all messages from a conversation in LLM format
func GetConversationMessages(conversationID int) ([]llm.Message, error) {
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
func GetConversationMessagesWithDetails(conversationID int) ([]Message, error) {
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
func DeleteConversation(convID int) error {
	db := GetDB()

	query := `DELETE FROM conversations WHERE id = $1`
	_, err := db.Exec(query, convID)
	if err != nil {
		return fmt.Errorf("error deleting conversation: %w", err)
	}

	log.Printf("[DB] Deleted conversation: %d", convID)
	return nil
}

// ClearConversationHistory clears all messages from a conversation
func ClearConversationHistory(convID int) error {
	db := GetDB()

	query := `DELETE FROM messages WHERE conversation_id = $1`
	_, err := db.Exec(query, convID)
	if err != nil {
		return fmt.Errorf("error clearing conversation history: %w", err)
	}

	log.Printf("[DB] Cleared history for conversation: %d", convID)
	return nil
}
