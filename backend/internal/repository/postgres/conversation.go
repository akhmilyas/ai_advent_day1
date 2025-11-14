package postgres

import (
	"chat-app/internal/logger"
	"chat-app/internal/repository/db"
	"chat-app/internal/service/llm"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CreateConversation creates a new conversation for a user
func (p *PostgresDB) CreateConversation(userID string, title string, responseFormat string, responseSchema string) (*db.Conversation, error) {
	conn := p.conn

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

	err := conn.QueryRow(query, convID, userID, title, responseFormat, responseSchema).Scan(&convID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("error creating conversation: %w", err)
	}

	logger.Log.WithFields(logrus.Fields{"conversation_id": convID, "user_id": userID, "format": responseFormat}).Info("Created new conversation")

	return &db.Conversation{
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
func (p *PostgresDB) GetConversationsByUser(userID string) ([]db.Conversation, error) {
	conn := p.conn

	query := `
	SELECT id, user_id, title, COALESCE(response_format, 'text'), COALESCE(response_schema, ''), created_at, updated_at
	FROM conversations
	WHERE user_id = $1
	ORDER BY updated_at DESC
	`

	rows, err := conn.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying conversations: %w", err)
	}
	defer rows.Close()

	var conversations []db.Conversation
	for rows.Next() {
		var conv db.Conversation
		if err := rows.Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.ResponseFormat, &conv.ResponseSchema, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// GetConversation retrieves a specific conversation
func (p *PostgresDB) GetConversation(convID string) (*db.Conversation, error) {
	conn := p.conn

	var conv db.Conversation
	query := `
	SELECT id, user_id, title, COALESCE(response_format, 'text'), COALESCE(response_schema, ''), active_summary_id, created_at, updated_at
	FROM conversations
	WHERE id = $1
	`

	err := conn.QueryRow(query, convID).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.ResponseFormat, &conv.ResponseSchema, &conv.ActiveSummaryID, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error retrieving conversation: %w", err)
	}

	return &conv, nil
}

// AddMessage adds a message to a conversation
func (p *PostgresDB) AddMessage(conversationID string, role, content, model string, temperature *float64, provider string, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
	conn := p.conn

	msgID := uuid.New().String()
	var createdAt time.Time

	query := `
	INSERT INTO messages (id, conversation_id, role, content, model, temperature, provider, generation_id, prompt_tokens, completion_tokens, total_tokens, total_cost, latency, generation_time)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	RETURNING id, created_at
	`

	err := conn.QueryRow(query, msgID, conversationID, role, content, model, temperature, provider, generationID, promptTokens, completionTokens, totalTokens, totalCost, latency, generationTime).Scan(&msgID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("error adding message: %w", err)
	}

	// Update conversation updated_at timestamp
	updateQuery := `UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	if _, err := conn.Exec(updateQuery, conversationID); err != nil {
		logger.Log.WithError(err).Warn("Error updating conversation timestamp")
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
	logger.Log.WithFields(logrus.Fields{
		"conversation_id": conversationID,
		"provider": providerStr,
		"model": model,
		"temperature": tempStr,
		"tokens": tokensStr,
		"cost": costStr,
		"latency": latencyStr,
		"generation_time": genTimeStr,
	}).Debug("Added message to conversation")

	return &db.Message{
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
	conn := p.conn

	query := `
	SELECT role, content
	FROM messages
	WHERE conversation_id = $1
	ORDER BY created_at ASC
	`

	rows, err := conn.Query(query, conversationID)
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
func (p *PostgresDB) GetConversationMessagesWithDetails(conversationID string) ([]db.Message, error) {
	conn := p.conn

	query := `
	SELECT id, conversation_id, role, content, COALESCE(model, ''), temperature, COALESCE(provider, ''),
	       COALESCE(generation_id, ''), prompt_tokens, completion_tokens, total_tokens, total_cost, latency, generation_time, created_at
	FROM messages
	WHERE conversation_id = $1
	ORDER BY created_at ASC
	`

	rows, err := conn.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("error querying messages: %w", err)
	}
	defer rows.Close()

	var messages []db.Message
	for rows.Next() {
		var msg db.Message
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
	conn := p.conn

	query := `DELETE FROM conversations WHERE id = $1`
	_, err := conn.Exec(query, convID)
	if err != nil {
		return fmt.Errorf("error deleting conversation: %w", err)
	}

	logger.Log.WithField("conversation_id", convID).Info("Deleted conversation")
	return nil
}

// CreateSummary creates a new conversation summary
func (p *PostgresDB) CreateSummary(conversationID string, summaryContent string, summarizedUpToMessageID *string) (*db.ConversationSummary, error) {
	conn := p.conn

	summaryID := uuid.New().String()
	var createdAt time.Time

	query := `
	INSERT INTO conversation_summaries (id, conversation_id, summary_content, summarized_up_to_message_id, usage_count)
	VALUES ($1, $2, $3, $4, 0)
	RETURNING id, created_at
	`

	err := conn.QueryRow(query, summaryID, conversationID, summaryContent, summarizedUpToMessageID).Scan(&summaryID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("error creating summary: %w", err)
	}

	logger.Log.WithFields(logrus.Fields{"summary_id": summaryID, "conversation_id": conversationID}).Info("Created summary")

	return &db.ConversationSummary{
		ID:                      summaryID,
		ConversationID:          conversationID,
		SummaryContent:          summaryContent,
		SummarizedUpToMessageID: summarizedUpToMessageID,
		UsageCount:              0,
		CreatedAt:               createdAt,
	}, nil
}

// GetActiveSummary retrieves the most recent summary for a conversation
func (p *PostgresDB) GetActiveSummary(conversationID string) (*db.ConversationSummary, error) {
	conn := p.conn

	var summary db.ConversationSummary
	query := `
	SELECT id, conversation_id, summary_content, summarized_up_to_message_id, usage_count, created_at
	FROM conversation_summaries
	WHERE conversation_id = $1
	ORDER BY created_at DESC
	LIMIT 1
	`

	err := conn.QueryRow(query, conversationID).Scan(
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

	logger.Log.WithFields(logrus.Fields{
		"summary_id":      summary.ID,
		"created_at":      summary.CreatedAt.Format(time.RFC3339),
		"usage_count":     summary.UsageCount,
		"conversation_id": conversationID,
	}).Debug("Retrieved most recent summary")

	return &summary, nil
}

// GetAllSummaries retrieves all summaries for a conversation in chronological order
func (p *PostgresDB) GetAllSummaries(conversationID string) ([]db.ConversationSummary, error) {
	conn := p.conn

	query := `
	SELECT id, conversation_id, summary_content, summarized_up_to_message_id, usage_count, created_at
	FROM conversation_summaries
	WHERE conversation_id = $1
	ORDER BY created_at ASC
	`

	rows, err := conn.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("error querying summaries: %w", err)
	}
	defer rows.Close()

	var summaries []db.ConversationSummary
	for rows.Next() {
		var summary db.ConversationSummary
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

	logger.Log.WithFields(logrus.Fields{"count": len(summaries), "conversation_id": conversationID}).Debug("Retrieved summaries")
	return summaries, nil
}

// UpdateConversationActiveSummary updates the active summary for a conversation
func (p *PostgresDB) UpdateConversationActiveSummary(conversationID string, summaryID string) error {
	conn := p.conn

	query := `UPDATE conversations SET active_summary_id = $1 WHERE id = $2`
	_, err := conn.Exec(query, summaryID, conversationID)
	if err != nil {
		return fmt.Errorf("error updating active summary: %w", err)
	}

	logger.Log.WithFields(logrus.Fields{"conversation_id": conversationID, "summary_id": summaryID}).Info("Updated active summary")
	return nil
}

// IncrementSummaryUsageCount increments the usage count for a summary
func (p *PostgresDB) IncrementSummaryUsageCount(summaryID string) error {
	conn := p.conn

	query := `UPDATE conversation_summaries SET usage_count = usage_count + 1 WHERE id = $1`
	_, err := conn.Exec(query, summaryID)
	if err != nil {
		return fmt.Errorf("error incrementing summary usage count: %w", err)
	}

	logger.Log.WithField("summary_id", summaryID).Debug("Incremented usage count for summary")
	return nil
}

// GetMessagesAfterMessage retrieves all messages after a specific message ID in a conversation
func (p *PostgresDB) GetMessagesAfterMessage(conversationID string, afterMessageID string) ([]llm.Message, error) {
	conn := p.conn

	query := `
	SELECT role, content
	FROM messages
	WHERE conversation_id = $1 AND created_at > (
		SELECT created_at FROM messages WHERE id = $2
	)
	ORDER BY created_at ASC
	`

	rows, err := conn.Query(query, conversationID, afterMessageID)
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
	conn := p.conn

	var messageID string
	query := `
	SELECT id
	FROM messages
	WHERE conversation_id = $1
	ORDER BY created_at DESC
	LIMIT 1
	`

	err := conn.QueryRow(query, conversationID).Scan(&messageID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // No messages yet
		}
		return nil, fmt.Errorf("error getting last message ID: %w", err)
	}

	return &messageID, nil
}

