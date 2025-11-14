package conversation

import (
	"chat-app/internal/repository/db"
	"fmt"
)

// ConversationWithSummary combines conversation info with its active summary
type ConversationWithSummary struct {
	ID                      string
	Title                   string
	ResponseFormat          string
	ResponseSchema          string
	SummarizedUpToMessageID *string
	CreatedAt               string
	UpdatedAt               string
}

// ConversationService handles the business logic for conversation management
type ConversationService struct {
	db db.Database
}

// NewConversationService creates a new ConversationService
func NewConversationService(database db.Database) *ConversationService {
	return &ConversationService{
		db: database,
	}
}

// GetUserConversations retrieves all conversations for a user with their active summaries
func (s *ConversationService) GetUserConversations(userID string) ([]ConversationWithSummary, error) {
	// Get all conversations for user
	conversations, err := s.db.GetConversationsByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conversations: %w", err)
	}

	// Convert to response format and fetch active summaries
	result := make([]ConversationWithSummary, 0, len(conversations))
	for _, conv := range conversations {
		// Get active summary for this conversation if it exists
		var summarizedUpToMsgID *string
		if summary, err := s.db.GetActiveSummary(conv.ID); err == nil && summary != nil {
			summarizedUpToMsgID = summary.SummarizedUpToMessageID
		}

		result = append(result, ConversationWithSummary{
			ID:                      conv.ID,
			Title:                   conv.Title,
			ResponseFormat:          conv.ResponseFormat,
			ResponseSchema:          conv.ResponseSchema,
			SummarizedUpToMessageID: summarizedUpToMsgID,
			CreatedAt:               conv.CreatedAt.String(),
			UpdatedAt:               conv.UpdatedAt.String(),
		})
	}

	return result, nil
}

// GetConversationMessages retrieves all messages from a specific conversation
func (s *ConversationService) GetConversationMessages(conversationID, userID string) ([]db.Message, error) {
	// Get conversation and verify ownership
	conversation, err := s.db.GetConversation(conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Verify user owns this conversation
	if conversation.UserID != userID {
		return nil, fmt.Errorf("unauthorized: user does not own this conversation")
	}

	// Get messages for conversation
	messages, err := s.db.GetConversationMessagesWithDetails(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}

	return messages, nil
}

// DeleteConversation deletes a conversation if the user owns it
func (s *ConversationService) DeleteConversation(conversationID, userID string) error {
	// Get conversation and verify ownership
	conversation, err := s.db.GetConversation(conversationID)
	if err != nil {
		return fmt.Errorf("conversation not found: %w", err)
	}

	// Verify user owns this conversation
	if conversation.UserID != userID {
		return fmt.Errorf("unauthorized: user does not own this conversation")
	}

	// Delete the conversation
	if err := s.db.DeleteConversation(conversationID); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}
