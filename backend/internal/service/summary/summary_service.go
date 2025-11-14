package summary

import (
	"chat-app/internal/app"
	"chat-app/internal/logger"
	"chat-app/internal/repository/db"
	"chat-app/internal/service/llm"
	"fmt"
)

// SummarizeRequest contains the parameters for summarization
type SummarizeRequest struct {
	ConversationID string
	UserID         string
	Model          string
	Temperature    *float64
}

// SummarizeResponse contains the result of summarization
type SummarizeResponse struct {
	Summary             string
	SummarizedUpToMsgID string
	ConversationID      string
	IsNewSummary        bool // True if new summary was created, false if existing returned
}

// SummaryService handles the business logic for conversation summarization
type SummaryService struct {
	db          db.Database
	config      *app.Config
	llmProvider llm.LLMProvider
}

// NewSummaryService creates a new SummaryService
func NewSummaryService(database db.Database, config *app.Config) *SummaryService {
	// Create LLM provider with configuration
	llmProvider := llm.NewOpenRouterProvider(
		&config.AppConfig.LLM,
		config.AppConfig.Models,
	)

	return &SummaryService{
		db:          database,
		config:      config,
		llmProvider: llmProvider,
	}
}

// SummarizeConversation creates a summary of a conversation
func (s *SummaryService) SummarizeConversation(req SummarizeRequest) (*SummarizeResponse, error) {
	// Get conversation and verify ownership
	conversation, err := s.db.GetConversation(req.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Verify user owns this conversation
	if conversation.UserID != req.UserID {
		return nil, fmt.Errorf("unauthorized: user does not own this conversation")
	}

	// Validate model if provided
	if err := s.validateModel(req.Model); err != nil {
		return nil, fmt.Errorf("invalid model: %w", err)
	}

	// Check if there's an existing active summary
	activeSummary, err := s.db.GetActiveSummary(req.ConversationID)
	if err != nil {
		activeSummary = nil // Treat error as no summary
	}

	// Check if we should create a new summary
	if !s.shouldCreateNewSummary(activeSummary) {
		// Summary exists but hasn't been used enough yet - return existing summary
		logger.Log.WithField("usage_count", activeSummary.UsageCount).Info("Active summary exists, not creating new summary")
		return &SummarizeResponse{
			Summary:             activeSummary.SummaryContent,
			SummarizedUpToMsgID: *activeSummary.SummarizedUpToMessageID,
			ConversationID:      req.ConversationID,
			IsNewSummary:        false,
		}, nil
	}

	// Build messages to summarize based on summary state
	messagesToSummarize, lastMessageID, err := s.buildSummarizationInput(req.ConversationID, activeSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to build summarization input: %w", err)
	}

	// Get summarization prompt from config
	summarizationPrompt := s.config.AppConfig.LLM.SummarizationPrompt
	if summarizationPrompt == "" {
		summarizationPrompt = `You are a conversation summarizer. Your task is to create a concise, comprehensive summary of the conversation that captures:
1. The main topics discussed
2. Key questions asked and answers provided
3. Important decisions or conclusions reached
4. Any action items or next steps mentioned

Format the summary in a clear, structured way that can be used as context for continuing the conversation. Keep the summary focused and avoid unnecessary details while preserving essential information.`
	}

	// Call LLM to generate summary (using ChatForSummarization to avoid default system prompt)
	logger.Log.WithField("message_count", len(messagesToSummarize)).Info("Calling LLM to generate summary")
	summaryContent, err := s.llmProvider.ChatForSummarization(messagesToSummarize, summarizationPrompt, req.Model, req.Temperature)
	if err != nil {
		return nil, fmt.Errorf("LLM error during summarization: %w", err)
	}

	logger.Log.WithField("summary_chars", len(summaryContent)).Info("Generated summary")

	// Create new summary in database
	summary, err := s.db.CreateSummary(req.ConversationID, summaryContent, lastMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to save summary: %w", err)
	}

	// Update conversation to use this new summary
	if err := s.db.UpdateConversationActiveSummary(req.ConversationID, summary.ID); err != nil {
		return nil, fmt.Errorf("failed to update active summary: %w", err)
	}

	return &SummarizeResponse{
		Summary:             summaryContent,
		SummarizedUpToMsgID: *lastMessageID,
		ConversationID:      req.ConversationID,
		IsNewSummary:        true,
	}, nil
}

// GetAllSummaries retrieves all summaries for a conversation
func (s *SummaryService) GetAllSummaries(conversationID, userID string) ([]db.ConversationSummary, error) {
	// Get conversation and verify ownership
	conversation, err := s.db.GetConversation(conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Verify user owns this conversation
	if conversation.UserID != userID {
		return nil, fmt.Errorf("unauthorized: user does not own this conversation")
	}

	// Get all summaries for conversation
	summaries, err := s.db.GetAllSummaries(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve summaries: %w", err)
	}

	return summaries, nil
}

// shouldCreateNewSummary determines if a new summary should be created
// Returns true if there's no existing summary or if the existing summary has been used 2+ times
func (s *SummaryService) shouldCreateNewSummary(summary *db.ConversationSummary) bool {
	return summary == nil || summary.UsageCount >= 2
}

// validateModel checks if the provided model ID is valid
func (s *SummaryService) validateModel(modelID string) error {
	if modelID != "" && !s.config.ModelsConfig().IsValidModel(modelID) {
		return fmt.Errorf("invalid model specified")
	}
	return nil
}

// getAllMessagesForSummary retrieves all messages for a conversation to create the first summary
func (s *SummaryService) getAllMessagesForSummary(convID string) ([]llm.Message, *string, error) {
	messages, err := s.db.GetConversationMessages(convID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting conversation messages: %w", err)
	}

	lastMessageID, err := s.db.GetLastMessageID(convID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting last message ID: %w", err)
	}

	return messages, lastMessageID, nil
}

// getIncrementalSummaryInput builds input for re-summarization using existing summary + new messages
func (s *SummaryService) getIncrementalSummaryInput(convID string, summary *db.ConversationSummary) ([]llm.Message, *string, error) {
	// Start with the old summary as context
	messages := []llm.Message{
		{Role: "assistant", Content: fmt.Sprintf("Previous summary:\n%s", summary.SummaryContent)},
	}

	// Get messages after the last summarized message
	if summary.SummarizedUpToMessageID != nil {
		newMessages, err := s.db.GetMessagesAfterMessage(convID, *summary.SummarizedUpToMessageID)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting messages after last summarized: %w", err)
		}
		messages = append(messages, newMessages...)
	}

	// Get the last message ID
	lastMessageID, err := s.db.GetLastMessageID(convID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting last message ID: %w", err)
	}

	return messages, lastMessageID, nil
}

// buildSummarizationInput determines the messages to summarize based on existing summary state
func (s *SummaryService) buildSummarizationInput(convID string, summary *db.ConversationSummary) ([]llm.Message, *string, error) {
	if summary == nil {
		logger.Log.Info("No active summary found, summarizing all messages")
		return s.getAllMessagesForSummary(convID)
	}

	logger.Log.WithField("usage_count", summary.UsageCount).Info("Creating new summary from active summary")
	return s.getIncrementalSummaryInput(convID, summary)
}
