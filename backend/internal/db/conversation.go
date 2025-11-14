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

// CreateConversation creates a new conversation for a user
func (p *PostgresDB) CreateConversation(userID string, title string, responseFormat string, responseSchema string) (*Conversation, error) {
	db := p.db

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
func (p *PostgresDB) GetConversationsByUser(userID string) ([]Conversation, error) {
	db := p.db

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
func (p *PostgresDB) GetConversation(convID string) (*Conversation, error) {
	db := p.db

	var conv Conversation
	query := `
	SELECT id, user_id, title, COALESCE(response_format, 'text'), COALESCE(response_schema, ''), active_summary_id, created_at, updated_at
	FROM conversations
	WHERE id = $1
	`

	err := db.QueryRow(query, convID).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.ResponseFormat, &conv.ResponseSchema, &conv.ActiveSummaryID, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error retrieving conversation: %w", err)
	}

	return &conv, nil
}

// AddMessage adds a message to a conversation
func (p *PostgresDB) AddMessage(conversationID string, role, content, model string, temperature *float64, provider string, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*Message, error) {
	db := p.db

	msgID := uuid.New().String()
	var createdAt time.Time

	query := `
	INSERT INTO messages (id, conversation_id, role, content, model, temperature, provider, generation_id, prompt_tokens, completion_tokens, total_tokens, total_cost, latency, generation_time)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	RETURNING id, created_at
	`

	err := db.QueryRow(query, msgID, conversationID, role, content, model, temperature, provider, generationID, promptTokens, completionTokens, totalTokens, totalCost, latency, generationTime).Scan(&msgID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("error adding message: %w", err)
	}

	// Update conversation updated_at timestamp
	updateQuery := `UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	if _, err := db.Exec(updateQuery, conversationID); err != nil {
		log.Printf("[DB] Warning: error updating conversation timestamp: %v", err)
	}

	tempStr := "nil"
	if temperature != nil {
		tempStr = fmt.Sprintf("%.2f", *temperature)
	}
	tokensStr := "nil"
	if totalTokens != nil {
		tokensStr = fmt.Sprintf("%d", *totalTokens)
	}
	costStr := "nil"
	if totalCost != nil {
		costStr = fmt.Sprintf("$%.6f", *totalCost)
	}
	latencyStr := "nil"
	if latency != nil {
		latencyStr = fmt.Sprintf("%dms", *latency)
	}
	genTimeStr := "nil"
	if generationTime != nil {
		genTimeStr = fmt.Sprintf("%dms", *generationTime)
	}
	providerStr := provider
	if providerStr == "" {
		providerStr = "unknown"
	}
	log.Printf("[DB] Added message to conversation %s with provider %s, model %s, temperature %s, tokens %s, cost %s, latency %s, generation_time %s", conversationID, providerStr, model, tempStr, tokensStr, costStr, latencyStr, genTimeStr)

	return &Message{
		ID:               msgID,
		ConversationID:   conversationID,
		Role:             role,
		Content:          content,
		Model:            model,
		Temperature:      temperature,
		Provider:         provider,
		GenerationID:     generationID,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		TotalCost:        totalCost,
		Latency:          latency,
		GenerationTime:   generationTime,
		CreatedAt:        createdAt,
	}, nil
}

// GetConversationMessages retrieves all messages from a conversation in LLM format
func (p *PostgresDB) GetConversationMessages(conversationID string) ([]llm.Message, error) {
	db := p.db

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
func (p *PostgresDB) GetConversationMessagesWithDetails(conversationID string) ([]Message, error) {
	db := p.db

	query := `
	SELECT id, conversation_id, role, content, COALESCE(model, ''), temperature, COALESCE(provider, ''),
	       COALESCE(generation_id, ''), prompt_tokens, completion_tokens, total_tokens, total_cost, latency, generation_time, created_at
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
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.Model, &msg.Temperature, &msg.Provider,
			&msg.GenerationID, &msg.PromptTokens, &msg.CompletionTokens, &msg.TotalTokens, &msg.TotalCost, &msg.Latency, &msg.GenerationTime, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// DeleteConversation deletes a conversation and all its messages
func (p *PostgresDB) DeleteConversation(convID string) error {
	db := p.db

	query := `DELETE FROM conversations WHERE id = $1`
	_, err := db.Exec(query, convID)
	if err != nil {
		return fmt.Errorf("error deleting conversation: %w", err)
	}

	log.Printf("[DB] Deleted conversation: %s", convID)
	return nil
}

// CreateSummary creates a new conversation summary
func (p *PostgresDB) CreateSummary(conversationID string, summaryContent string, summarizedUpToMessageID *string) (*ConversationSummary, error) {
	db := p.db

	summaryID := uuid.New().String()
	var createdAt time.Time

	query := `
	INSERT INTO conversation_summaries (id, conversation_id, summary_content, summarized_up_to_message_id, usage_count)
	VALUES ($1, $2, $3, $4, 0)
	RETURNING id, created_at
	`

	err := db.QueryRow(query, summaryID, conversationID, summaryContent, summarizedUpToMessageID).Scan(&summaryID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("error creating summary: %w", err)
	}

	log.Printf("[DB] Created summary %s for conversation %s", summaryID, conversationID)

	return &ConversationSummary{
		ID:                      summaryID,
		ConversationID:          conversationID,
		SummaryContent:          summaryContent,
		SummarizedUpToMessageID: summarizedUpToMessageID,
		UsageCount:              0,
		CreatedAt:               createdAt,
	}, nil
}

// GetActiveSummary retrieves the most recent summary for a conversation
func (p *PostgresDB) GetActiveSummary(conversationID string) (*ConversationSummary, error) {
	db := p.db

	var summary ConversationSummary
	query := `
	SELECT id, conversation_id, summary_content, summarized_up_to_message_id, usage_count, created_at
	FROM conversation_summaries
	WHERE conversation_id = $1
	ORDER BY created_at DESC
	LIMIT 1
	`

	err := db.QueryRow(query, conversationID).Scan(
		&summary.ID,
		&summary.ConversationID,
		&summary.SummaryContent,
		&summary.SummarizedUpToMessageID,
		&summary.UsageCount,
		&summary.CreatedAt,
	)
	if err != nil {
		return nil, err // Return nil if no summary exists
	}

	log.Printf("[DB] Retrieved most recent summary %s (created: %s, usage_count: %d) for conversation %s",
		summary.ID, summary.CreatedAt.Format(time.RFC3339), summary.UsageCount, conversationID)

	return &summary, nil
}

// GetAllSummaries retrieves all summaries for a conversation in chronological order
func (p *PostgresDB) GetAllSummaries(conversationID string) ([]ConversationSummary, error) {
	db := p.db

	query := `
	SELECT id, conversation_id, summary_content, summarized_up_to_message_id, usage_count, created_at
	FROM conversation_summaries
	WHERE conversation_id = $1
	ORDER BY created_at ASC
	`

	rows, err := db.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("error querying summaries: %w", err)
	}
	defer rows.Close()

	var summaries []ConversationSummary
	for rows.Next() {
		var summary ConversationSummary
		if err := rows.Scan(
			&summary.ID,
			&summary.ConversationID,
			&summary.SummaryContent,
			&summary.SummarizedUpToMessageID,
			&summary.UsageCount,
			&summary.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning summary: %w", err)
		}
		summaries = append(summaries, summary)
	}

	log.Printf("[DB] Retrieved %d summaries for conversation %s", len(summaries), conversationID)
	return summaries, nil
}

// UpdateConversationActiveSummary updates the active summary for a conversation
func (p *PostgresDB) UpdateConversationActiveSummary(conversationID string, summaryID string) error {
	db := p.db

	query := `UPDATE conversations SET active_summary_id = $1 WHERE id = $2`
	_, err := db.Exec(query, summaryID, conversationID)
	if err != nil {
		return fmt.Errorf("error updating active summary: %w", err)
	}

	log.Printf("[DB] Updated active summary for conversation %s to %s", conversationID, summaryID)
	return nil
}

// IncrementSummaryUsageCount increments the usage count for a summary
func (p *PostgresDB) IncrementSummaryUsageCount(summaryID string) error {
	db := p.db

	query := `UPDATE conversation_summaries SET usage_count = usage_count + 1 WHERE id = $1`
	_, err := db.Exec(query, summaryID)
	if err != nil {
		return fmt.Errorf("error incrementing summary usage count: %w", err)
	}

	log.Printf("[DB] Incremented usage count for summary %s", summaryID)
	return nil
}

// GetMessagesAfterMessage retrieves all messages after a specific message ID in a conversation
func (p *PostgresDB) GetMessagesAfterMessage(conversationID string, afterMessageID string) ([]llm.Message, error) {
	db := p.db

	query := `
	SELECT role, content
	FROM messages
	WHERE conversation_id = $1 AND created_at > (
		SELECT created_at FROM messages WHERE id = $2
	)
	ORDER BY created_at ASC
	`

	rows, err := db.Query(query, conversationID, afterMessageID)
	if err != nil {
		return nil, fmt.Errorf("error querying messages after message: %w", err)
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

// GetLastMessageID retrieves the ID of the last message in a conversation
func (p *PostgresDB) GetLastMessageID(conversationID string) (*string, error) {
	db := p.db

	var messageID string
	query := `
	SELECT id
	FROM messages
	WHERE conversation_id = $1
	ORDER BY created_at DESC
	LIMIT 1
	`

	err := db.QueryRow(query, conversationID).Scan(&messageID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // No messages yet
		}
		return nil, fmt.Errorf("error getting last message ID: %w", err)
	}

	return &messageID, nil
}

// Standalone function wrappers for backwards compatibility
// These delegate to the PostgresDB methods

func CreateConversation(userID string, title string, responseFormat string, responseSchema string) (*Conversation, error) {
	db := NewPostgresDB()
	return db.CreateConversation(userID, title, responseFormat, responseSchema)
}

func GetConversationsByUser(userID string) ([]Conversation, error) {
	db := NewPostgresDB()
	return db.GetConversationsByUser(userID)
}

func GetConversation(convID string) (*Conversation, error) {
	db := NewPostgresDB()
	return db.GetConversation(convID)
}

func AddMessage(conversationID string, role, content, model string, temperature *float64, provider string, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*Message, error) {
	db := NewPostgresDB()
	return db.AddMessage(conversationID, role, content, model, temperature, provider, generationID, promptTokens, completionTokens, totalTokens, totalCost, latency, generationTime)
}

func GetConversationMessages(conversationID string) ([]llm.Message, error) {
	db := NewPostgresDB()
	return db.GetConversationMessages(conversationID)
}

func GetConversationMessagesWithDetails(conversationID string) ([]Message, error) {
	db := NewPostgresDB()
	return db.GetConversationMessagesWithDetails(conversationID)
}

func DeleteConversation(convID string) error {
	db := NewPostgresDB()
	return db.DeleteConversation(convID)
}

func CreateSummary(conversationID string, summaryContent string, summarizedUpToMessageID *string) (*ConversationSummary, error) {
	db := NewPostgresDB()
	return db.CreateSummary(conversationID, summaryContent, summarizedUpToMessageID)
}

func GetActiveSummary(conversationID string) (*ConversationSummary, error) {
	db := NewPostgresDB()
	return db.GetActiveSummary(conversationID)
}

func GetAllSummaries(conversationID string) ([]ConversationSummary, error) {
	db := NewPostgresDB()
	return db.GetAllSummaries(conversationID)
}

func UpdateConversationActiveSummary(conversationID string, summaryID string) error {
	db := NewPostgresDB()
	return db.UpdateConversationActiveSummary(conversationID, summaryID)
}

func IncrementSummaryUsageCount(summaryID string) error {
	db := NewPostgresDB()
	return db.IncrementSummaryUsageCount(summaryID)
}

func GetMessagesAfterMessage(conversationID string, afterMessageID string) ([]llm.Message, error) {
	db := NewPostgresDB()
	return db.GetMessagesAfterMessage(conversationID, afterMessageID)
}

func GetLastMessageID(conversationID string) (*string, error) {
	db := NewPostgresDB()
	return db.GetLastMessageID(conversationID)
}
