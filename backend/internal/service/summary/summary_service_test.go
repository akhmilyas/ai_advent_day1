package summary

import (
	"chat-app/internal/repository/db"
	"chat-app/internal/service/llm"
	"chat-app/internal/testutil"
	"errors"
	"testing"
	"time"
)

func TestNewSummaryService(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockConfig := testutil.NewMockConfig()

	service := NewSummaryService(mockDB, mockConfig)

	if service == nil {
		t.Fatal("NewSummaryService returned nil")
	}

	if service.db == nil {
		t.Error("SummaryService database not set")
	}

	if service.config == nil {
		t.Error("SummaryService config not set")
	}

	if service.llmProvider == nil {
		t.Error("SummaryService llmProvider not set")
	}
}

func TestSummarizeConversation_ConversationNotFound(t *testing.T) {
	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return nil, errors.New("conversation not found")
		},
	}
	mockConfig := testutil.NewMockConfig()

	service := NewSummaryService(mockDB, mockConfig)
	_, err := service.SummarizeConversation(SummarizeRequest{
		ConversationID: "conv-123",
		UserID:         "user-123",
	})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "conversation not found") {
		t.Errorf("Expected error to contain 'conversation not found', got: %s", err.Error())
	}
}

func TestSummarizeConversation_Unauthorized(t *testing.T) {
	now := time.Now()
	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return &db.Conversation{
				ID:             "conv-123",
				UserID:         "user-999", // Different user
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	}
	mockConfig := testutil.NewMockConfig()

	service := NewSummaryService(mockDB, mockConfig)
	_, err := service.SummarizeConversation(SummarizeRequest{
		ConversationID: "conv-123",
		UserID:         "user-123",
	})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "unauthorized") {
		t.Errorf("Expected error to contain 'unauthorized', got: %s", err.Error())
	}
}

func TestSummarizeConversation_InvalidModel(t *testing.T) {
	now := time.Now()
	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return &db.Conversation{
				ID:             "conv-123",
				UserID:         "user-123",
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
		GetActiveSummaryFunc: func(conversationID string) (*db.ConversationSummary, error) {
			return nil, errors.New("no summary")
		},
	}
	mockConfig := testutil.NewMockConfig()

	service := NewSummaryService(mockDB, mockConfig)
	_, err := service.SummarizeConversation(SummarizeRequest{
		ConversationID: "conv-123",
		UserID:         "user-123",
		Model:          "invalid-model",
	})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "invalid model") {
		t.Errorf("Expected error to contain 'invalid model', got: %s", err.Error())
	}
}

func TestSummarizeConversation_ExistingSummaryNotUsedEnough(t *testing.T) {
	now := time.Now()
	summarizedMsgID := "msg-50"
	existingSummary := &db.ConversationSummary{
		ID:                      "summary-1",
		ConversationID:          "conv-123",
		SummaryContent:          "Existing summary content",
		SummarizedUpToMessageID: &summarizedMsgID,
		UsageCount:              1, // Less than threshold of 2
		CreatedAt:               now,
	}

	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return &db.Conversation{
				ID:             "conv-123",
				UserID:         "user-123",
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
		GetActiveSummaryFunc: func(conversationID string) (*db.ConversationSummary, error) {
			return existingSummary, nil
		},
	}
	mockConfig := testutil.NewMockConfig()

	service := NewSummaryService(mockDB, mockConfig)
	resp, err := service.SummarizeConversation(SummarizeRequest{
		ConversationID: "conv-123",
		UserID:         "user-123",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.IsNewSummary {
		t.Error("Expected IsNewSummary to be false")
	}

	if resp.Summary != existingSummary.SummaryContent {
		t.Errorf("Expected summary content %s, got %s", existingSummary.SummaryContent, resp.Summary)
	}

	if resp.SummarizedUpToMsgID != summarizedMsgID {
		t.Errorf("Expected summarized up to %s, got %s", summarizedMsgID, resp.SummarizedUpToMsgID)
	}
}

func TestSummarizeConversation_FirstSummary_Success(t *testing.T) {
	now := time.Now()
	lastMsgID := "msg-100"
	llmSummary := "This is a generated summary of the conversation"

	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return &db.Conversation{
				ID:             "conv-123",
				UserID:         "user-123",
				Title:          "Test Conversation",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
		GetActiveSummaryFunc: func(conversationID string) (*db.ConversationSummary, error) {
			return nil, errors.New("no summary")
		},
		GetConversationMessagesFunc: func(conversationID string) ([]llm.Message, error) {
			return []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
				{Role: "user", Content: "How are you?"},
				{Role: "assistant", Content: "I'm doing well, thanks!"},
			}, nil
		},
		GetLastMessageIDFunc: func(conversationID string) (*string, error) {
			return &lastMsgID, nil
		},
		CreateSummaryFunc: func(conversationID, summaryContent string, summarizedUpToMessageID *string) (*db.ConversationSummary, error) {
			if summaryContent != llmSummary {
				t.Errorf("CreateSummary called with wrong content: got %s, want %s", summaryContent, llmSummary)
			}
			if summarizedUpToMessageID == nil || *summarizedUpToMessageID != lastMsgID {
				t.Error("CreateSummary called with wrong summarizedUpToMessageID")
			}
			return &db.ConversationSummary{
				ID:                      "summary-new",
				ConversationID:          conversationID,
				SummaryContent:          summaryContent,
				SummarizedUpToMessageID: summarizedUpToMessageID,
				UsageCount:              0,
				CreatedAt:               now,
			}, nil
		},
		UpdateConversationActiveSummaryFunc: func(conversationID, summaryID string) error {
			if conversationID != "conv-123" {
				t.Errorf("UpdateConversationActiveSummary called with wrong conversationID: %s", conversationID)
			}
			if summaryID != "summary-new" {
				t.Errorf("UpdateConversationActiveSummary called with wrong summaryID: %s", summaryID)
			}
			return nil
		},
	}

	mockConfig := testutil.NewMockConfig()

	// Create a custom service with mock LLM provider
	mockLLM := &testutil.MockLLMProvider{
		ChatForSummarizationFunc: func(messages []llm.Message, summarizationPrompt string, modelOverride string, temperature *float64) (string, error) {
			if len(messages) != 4 {
				t.Errorf("Expected 4 messages, got %d", len(messages))
			}
			if summarizationPrompt == "" {
				t.Error("Summarization prompt is empty")
			}
			return llmSummary, nil
		},
		GetDefaultModelFunc: func() string {
			return "test-model"
		},
	}

	service := &SummaryService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	resp, err := service.SummarizeConversation(SummarizeRequest{
		ConversationID: "conv-123",
		UserID:         "user-123",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !resp.IsNewSummary {
		t.Error("Expected IsNewSummary to be true")
	}

	if resp.Summary != llmSummary {
		t.Errorf("Expected summary %s, got %s", llmSummary, resp.Summary)
	}

	if resp.SummarizedUpToMsgID != lastMsgID {
		t.Errorf("Expected summarized up to %s, got %s", lastMsgID, resp.SummarizedUpToMsgID)
	}

	if resp.ConversationID != "conv-123" {
		t.Errorf("Expected conversation ID conv-123, got %s", resp.ConversationID)
	}
}

func TestSummarizeConversation_ProgressiveResummarization(t *testing.T) {
	now := time.Now()
	oldMsgID := "msg-50"
	newMsgID := "msg-100"
	oldSummary := "Old summary content"
	newSummary := "Updated summary with new messages"

	existingSummary := &db.ConversationSummary{
		ID:                      "summary-1",
		ConversationID:          "conv-123",
		SummaryContent:          oldSummary,
		SummarizedUpToMessageID: &oldMsgID,
		UsageCount:              2, // Meets threshold for re-summarization
		CreatedAt:               now,
	}

	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return &db.Conversation{
				ID:             "conv-123",
				UserID:         "user-123",
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
		GetActiveSummaryFunc: func(conversationID string) (*db.ConversationSummary, error) {
			return existingSummary, nil
		},
		GetMessagesAfterMessageFunc: func(conversationID, afterMessageID string) ([]llm.Message, error) {
			if afterMessageID != oldMsgID {
				t.Errorf("GetMessagesAfterMessage called with wrong afterMessageID: got %s, want %s", afterMessageID, oldMsgID)
			}
			return []llm.Message{
				{Role: "user", Content: "New message 1"},
				{Role: "assistant", Content: "Response 1"},
			}, nil
		},
		GetLastMessageIDFunc: func(conversationID string) (*string, error) {
			return &newMsgID, nil
		},
		CreateSummaryFunc: func(conversationID, summaryContent string, summarizedUpToMessageID *string) (*db.ConversationSummary, error) {
			if summaryContent != newSummary {
				t.Errorf("CreateSummary called with wrong content: got %s, want %s", summaryContent, newSummary)
			}
			return &db.ConversationSummary{
				ID:                      "summary-2",
				ConversationID:          conversationID,
				SummaryContent:          summaryContent,
				SummarizedUpToMessageID: summarizedUpToMessageID,
				UsageCount:              0,
				CreatedAt:               time.Now(),
			}, nil
		},
		UpdateConversationActiveSummaryFunc: func(conversationID, summaryID string) error {
			if summaryID != "summary-2" {
				t.Errorf("UpdateConversationActiveSummary called with wrong summaryID: %s", summaryID)
			}
			return nil
		},
	}

	mockConfig := testutil.NewMockConfig()

	mockLLM := &testutil.MockLLMProvider{
		ChatForSummarizationFunc: func(messages []llm.Message, summarizationPrompt string, modelOverride string, temperature *float64) (string, error) {
			// Should have 3 messages: previous summary + 2 new messages
			if len(messages) != 3 {
				t.Errorf("Expected 3 messages (summary + new), got %d", len(messages))
			}
			// First message should contain the old summary
			if !contains(messages[0].Content, oldSummary) {
				t.Errorf("First message should contain old summary")
			}
			return newSummary, nil
		},
		GetDefaultModelFunc: func() string {
			return "test-model"
		},
	}

	service := &SummaryService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	resp, err := service.SummarizeConversation(SummarizeRequest{
		ConversationID: "conv-123",
		UserID:         "user-123",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !resp.IsNewSummary {
		t.Error("Expected IsNewSummary to be true")
	}

	if resp.Summary != newSummary {
		t.Errorf("Expected summary %s, got %s", newSummary, resp.Summary)
	}

	if resp.SummarizedUpToMsgID != newMsgID {
		t.Errorf("Expected summarized up to %s, got %s", newMsgID, resp.SummarizedUpToMsgID)
	}
}

func TestSummarizeConversation_LLMError(t *testing.T) {
	now := time.Now()
	lastMsgID := "msg-100"

	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return &db.Conversation{
				ID:             "conv-123",
				UserID:         "user-123",
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
		GetActiveSummaryFunc: func(conversationID string) (*db.ConversationSummary, error) {
			return nil, errors.New("no summary")
		},
		GetConversationMessagesFunc: func(conversationID string) ([]llm.Message, error) {
			return []llm.Message{
				{Role: "user", Content: "Hello"},
			}, nil
		},
		GetLastMessageIDFunc: func(conversationID string) (*string, error) {
			return &lastMsgID, nil
		},
	}

	mockConfig := testutil.NewMockConfig()

	mockLLM := &testutil.MockLLMProvider{
		ChatForSummarizationFunc: func(messages []llm.Message, summarizationPrompt string, modelOverride string, temperature *float64) (string, error) {
			return "", errors.New("LLM API error")
		},
	}

	service := &SummaryService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	_, err := service.SummarizeConversation(SummarizeRequest{
		ConversationID: "conv-123",
		UserID:         "user-123",
	})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "LLM error") {
		t.Errorf("Expected error to contain 'LLM error', got: %s", err.Error())
	}
}

func TestGetAllSummaries_Success(t *testing.T) {
	now := time.Now()
	msgID1 := "msg-50"
	msgID2 := "msg-100"

	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return &db.Conversation{
				ID:             "conv-123",
				UserID:         "user-123",
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
		GetAllSummariesFunc: func(conversationID string) ([]db.ConversationSummary, error) {
			return []db.ConversationSummary{
				{
					ID:                      "summary-1",
					ConversationID:          conversationID,
					SummaryContent:          "First summary",
					SummarizedUpToMessageID: &msgID1,
					UsageCount:              2,
					CreatedAt:               now,
				},
				{
					ID:                      "summary-2",
					ConversationID:          conversationID,
					SummaryContent:          "Second summary",
					SummarizedUpToMessageID: &msgID2,
					UsageCount:              0,
					CreatedAt:               now.Add(time.Hour),
				},
			}, nil
		},
	}

	mockConfig := testutil.NewMockConfig()
	service := NewSummaryService(mockDB, mockConfig)

	summaries, err := service.GetAllSummaries("conv-123", "user-123")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	if summaries[0].SummaryContent != "First summary" {
		t.Errorf("Expected first summary content 'First summary', got %s", summaries[0].SummaryContent)
	}

	if summaries[1].UsageCount != 0 {
		t.Errorf("Expected second summary usage count 0, got %d", summaries[1].UsageCount)
	}
}

func TestGetAllSummaries_Unauthorized(t *testing.T) {
	now := time.Now()

	mockDB := &testutil.MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return &db.Conversation{
				ID:             "conv-123",
				UserID:         "user-999", // Different user
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	}

	mockConfig := testutil.NewMockConfig()
	service := NewSummaryService(mockDB, mockConfig)

	_, err := service.GetAllSummaries("conv-123", "user-123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "unauthorized") {
		t.Errorf("Expected error to contain 'unauthorized', got: %s", err.Error())
	}
}

func TestShouldCreateNewSummary(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockConfig := testutil.NewMockConfig()
	service := NewSummaryService(mockDB, mockConfig)

	tests := []struct {
		name     string
		summary  *db.ConversationSummary
		expected bool
	}{
		{
			name:     "nil summary",
			summary:  nil,
			expected: true,
		},
		{
			name: "usage count 0",
			summary: &db.ConversationSummary{
				UsageCount: 0,
			},
			expected: false,
		},
		{
			name: "usage count 1",
			summary: &db.ConversationSummary{
				UsageCount: 1,
			},
			expected: false,
		},
		{
			name: "usage count 2 (threshold)",
			summary: &db.ConversationSummary{
				UsageCount: 2,
			},
			expected: true,
		},
		{
			name: "usage count 5 (above threshold)",
			summary: &db.ConversationSummary{
				UsageCount: 5,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.shouldCreateNewSummary(tt.summary)
			if result != tt.expected {
				t.Errorf("shouldCreateNewSummary() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
