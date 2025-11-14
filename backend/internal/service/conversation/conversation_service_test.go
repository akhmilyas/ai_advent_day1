package conversation

import (
	"chat-app/internal/repository/db"
	"chat-app/internal/service/llm"
	"errors"
	"testing"
	"time"
)

// MockDatabase is a mock implementation of db.Database for testing
type MockDatabase struct {
	// Conversation mocks
	GetConversationFunc       func(id string) (*db.Conversation, error)
	GetConversationsByUserFunc func(userID string) ([]db.Conversation, error)
	DeleteConversationFunc    func(id string) error
	GetActiveSummaryFunc      func(conversationID string) (*db.ConversationSummary, error)
	GetConversationMessagesWithDetailsFunc func(conversationID string) ([]db.Message, error)
}

// Implement db.Database interface methods (only those used by ConversationService)
func (m *MockDatabase) GetConversation(id string) (*db.Conversation, error) {
	if m.GetConversationFunc != nil {
		return m.GetConversationFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetConversationsByUser(userID string) ([]db.Conversation, error) {
	if m.GetConversationsByUserFunc != nil {
		return m.GetConversationsByUserFunc(userID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) DeleteConversation(id string) error {
	if m.DeleteConversationFunc != nil {
		return m.DeleteConversationFunc(id)
	}
	return errors.New("not implemented")
}

func (m *MockDatabase) GetActiveSummary(conversationID string) (*db.ConversationSummary, error) {
	if m.GetActiveSummaryFunc != nil {
		return m.GetActiveSummaryFunc(conversationID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetConversationMessagesWithDetails(conversationID string) ([]db.Message, error) {
	if m.GetConversationMessagesWithDetailsFunc != nil {
		return m.GetConversationMessagesWithDetailsFunc(conversationID)
	}
	return nil, errors.New("not implemented")
}

// Stub implementations for unused interface methods
func (m *MockDatabase) GetUserByUsername(username string) (*db.User, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) CreateUser(username, email, passwordHash string) (*db.User, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) CreateConversation(userID, title, format, schema string) (*db.Conversation, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) AddMessage(conversationID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetConversationMessages(conversationID string) ([]llm.Message, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetMessagesAfterMessage(conversationID, afterMessageID string) ([]llm.Message, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetLastMessageID(conversationID string) (*string, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) CreateSummary(conversationID, summaryContent string, summarizedUpToMessageID *string) (*db.ConversationSummary, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetAllSummaries(conversationID string) ([]db.ConversationSummary, error) {
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) IncrementSummaryUsageCount(summaryID string) error {
	return errors.New("not implemented")
}

func (m *MockDatabase) UpdateConversationActiveSummary(conversationID, summaryID string) error {
	return errors.New("not implemented")
}

// Tests

func TestNewConversationService(t *testing.T) {
	mockDB := &MockDatabase{}
	service := NewConversationService(mockDB)

	if service == nil {
		t.Fatal("NewConversationService returned nil")
	}

	// Service should be properly initialized (can't compare interface directly)
	if service.db == nil {
		t.Error("ConversationService database not set")
	}
}

func TestGetUserConversations_Success(t *testing.T) {
	now := time.Now()
	userID := "user-123"

	mockDB := &MockDatabase{
		GetConversationsByUserFunc: func(uid string) ([]db.Conversation, error) {
			if uid != userID {
				t.Errorf("GetConversationsByUser called with wrong userID: got %s, want %s", uid, userID)
			}
			return []db.Conversation{
				{
					ID:             "conv-1",
					UserID:         userID,
					Title:          "Test Conversation 1",
					ResponseFormat: "text",
					ResponseSchema: "",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             "conv-2",
					UserID:         userID,
					Title:          "Test Conversation 2",
					ResponseFormat: "json",
					ResponseSchema: "{\"type\":\"object\"}",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			}, nil
		},
		GetActiveSummaryFunc: func(conversationID string) (*db.ConversationSummary, error) {
			// No active summaries for these conversations
			return nil, errors.New("not found")
		},
	}

	service := NewConversationService(mockDB)
	conversations, err := service.GetUserConversations(userID)

	if err != nil {
		t.Fatalf("GetUserConversations returned error: %v", err)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(conversations))
	}

	// Verify first conversation
	if conversations[0].ID != "conv-1" {
		t.Errorf("First conversation ID: got %s, want conv-1", conversations[0].ID)
	}
	if conversations[0].Title != "Test Conversation 1" {
		t.Errorf("First conversation Title: got %s, want Test Conversation 1", conversations[0].Title)
	}
	if conversations[0].ResponseFormat != "text" {
		t.Errorf("First conversation ResponseFormat: got %s, want text", conversations[0].ResponseFormat)
	}
	if conversations[0].SummarizedUpToMessageID != nil {
		t.Error("First conversation should not have summarized message ID")
	}

	// Verify second conversation
	if conversations[1].ID != "conv-2" {
		t.Errorf("Second conversation ID: got %s, want conv-2", conversations[1].ID)
	}
	if conversations[1].ResponseFormat != "json" {
		t.Errorf("Second conversation ResponseFormat: got %s, want json", conversations[1].ResponseFormat)
	}
	if conversations[1].ResponseSchema != "{\"type\":\"object\"}" {
		t.Errorf("Second conversation ResponseSchema: got %s, want {\"type\":\"object\"}", conversations[1].ResponseSchema)
	}
}

func TestGetUserConversations_WithActiveSummary(t *testing.T) {
	now := time.Now()
	userID := "user-123"
	summarizedMsgID := "msg-100"

	mockDB := &MockDatabase{
		GetConversationsByUserFunc: func(uid string) ([]db.Conversation, error) {
			return []db.Conversation{
				{
					ID:             "conv-1",
					UserID:         userID,
					Title:          "Test Conversation",
					ResponseFormat: "text",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			}, nil
		},
		GetActiveSummaryFunc: func(conversationID string) (*db.ConversationSummary, error) {
			if conversationID == "conv-1" {
				return &db.ConversationSummary{
					ID:                      "summary-1",
					ConversationID:          "conv-1",
					SummaryContent:          "This is a summary",
					SummarizedUpToMessageID: &summarizedMsgID,
					UsageCount:              0,
					CreatedAt:               now,
				}, nil
			}
			return nil, errors.New("not found")
		},
	}

	service := NewConversationService(mockDB)
	conversations, err := service.GetUserConversations(userID)

	if err != nil {
		t.Fatalf("GetUserConversations returned error: %v", err)
	}

	if len(conversations) != 1 {
		t.Fatalf("Expected 1 conversation, got %d", len(conversations))
	}

	if conversations[0].SummarizedUpToMessageID == nil {
		t.Fatal("Expected conversation to have SummarizedUpToMessageID")
	}

	if *conversations[0].SummarizedUpToMessageID != summarizedMsgID {
		t.Errorf("SummarizedUpToMessageID: got %s, want %s", *conversations[0].SummarizedUpToMessageID, summarizedMsgID)
	}
}

func TestGetUserConversations_DatabaseError(t *testing.T) {
	mockDB := &MockDatabase{
		GetConversationsByUserFunc: func(uid string) ([]db.Conversation, error) {
			return nil, errors.New("database connection error")
		},
	}

	service := NewConversationService(mockDB)
	_, err := service.GetUserConversations("user-123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "failed to retrieve conversations") {
		t.Errorf("Expected error message to contain 'failed to retrieve conversations', got: %s", err.Error())
	}
}

func TestGetUserConversations_EmptyList(t *testing.T) {
	mockDB := &MockDatabase{
		GetConversationsByUserFunc: func(uid string) ([]db.Conversation, error) {
			return []db.Conversation{}, nil
		},
	}

	service := NewConversationService(mockDB)
	conversations, err := service.GetUserConversations("user-123")

	if err != nil {
		t.Fatalf("GetUserConversations returned error: %v", err)
	}

	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations, got %d", len(conversations))
	}
}

func TestGetConversationMessages_Success(t *testing.T) {
	now := time.Now()
	conversationID := "conv-123"
	userID := "user-123"

	mockDB := &MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			if id != conversationID {
				t.Errorf("GetConversation called with wrong ID: got %s, want %s", id, conversationID)
			}
			return &db.Conversation{
				ID:             conversationID,
				UserID:         userID,
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
		GetConversationMessagesWithDetailsFunc: func(convID string) ([]db.Message, error) {
			if convID != conversationID {
				t.Errorf("GetConversationMessagesWithDetails called with wrong ID: got %s, want %s", convID, conversationID)
			}
			return []db.Message{
				{
					ID:             "msg-1",
					ConversationID: conversationID,
					Role:           "user",
					Content:        "Hello",
					CreatedAt:      now,
				},
				{
					ID:             "msg-2",
					ConversationID: conversationID,
					Role:           "assistant",
					Content:        "Hi there!",
					Model:          "gpt-4",
					CreatedAt:      now,
				},
			}, nil
		},
	}

	service := NewConversationService(mockDB)
	messages, err := service.GetConversationMessages(conversationID, userID)

	if err != nil {
		t.Fatalf("GetConversationMessages returned error: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("First message role: got %s, want user", messages[0].Role)
	}
	if messages[1].Role != "assistant" {
		t.Errorf("Second message role: got %s, want assistant", messages[1].Role)
	}
}

func TestGetConversationMessages_ConversationNotFound(t *testing.T) {
	mockDB := &MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return nil, errors.New("conversation not found")
		},
	}

	service := NewConversationService(mockDB)
	_, err := service.GetConversationMessages("conv-123", "user-123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "conversation not found") {
		t.Errorf("Expected error message to contain 'conversation not found', got: %s", err.Error())
	}
}

func TestGetConversationMessages_Unauthorized(t *testing.T) {
	now := time.Now()

	mockDB := &MockDatabase{
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

	service := NewConversationService(mockDB)
	_, err := service.GetConversationMessages("conv-123", "user-123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "unauthorized") {
		t.Errorf("Expected error message to contain 'unauthorized', got: %s", err.Error())
	}
}

func TestGetConversationMessages_DatabaseError(t *testing.T) {
	now := time.Now()

	mockDB := &MockDatabase{
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
		GetConversationMessagesWithDetailsFunc: func(convID string) ([]db.Message, error) {
			return nil, errors.New("database error")
		},
	}

	service := NewConversationService(mockDB)
	_, err := service.GetConversationMessages("conv-123", "user-123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "failed to retrieve messages") {
		t.Errorf("Expected error message to contain 'failed to retrieve messages', got: %s", err.Error())
	}
}

func TestDeleteConversation_Success(t *testing.T) {
	now := time.Now()
	conversationID := "conv-123"
	userID := "user-123"
	deleted := false

	mockDB := &MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			if id != conversationID {
				t.Errorf("GetConversation called with wrong ID: got %s, want %s", id, conversationID)
			}
			return &db.Conversation{
				ID:             conversationID,
				UserID:         userID,
				Title:          "Test",
				ResponseFormat: "text",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
		DeleteConversationFunc: func(id string) error {
			if id != conversationID {
				t.Errorf("DeleteConversation called with wrong ID: got %s, want %s", id, conversationID)
			}
			deleted = true
			return nil
		},
	}

	service := NewConversationService(mockDB)
	err := service.DeleteConversation(conversationID, userID)

	if err != nil {
		t.Fatalf("DeleteConversation returned error: %v", err)
	}

	if !deleted {
		t.Error("DeleteConversation was not called")
	}
}

func TestDeleteConversation_ConversationNotFound(t *testing.T) {
	mockDB := &MockDatabase{
		GetConversationFunc: func(id string) (*db.Conversation, error) {
			return nil, errors.New("conversation not found")
		},
	}

	service := NewConversationService(mockDB)
	err := service.DeleteConversation("conv-123", "user-123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "conversation not found") {
		t.Errorf("Expected error message to contain 'conversation not found', got: %s", err.Error())
	}
}

func TestDeleteConversation_Unauthorized(t *testing.T) {
	now := time.Now()

	mockDB := &MockDatabase{
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

	service := NewConversationService(mockDB)
	err := service.DeleteConversation("conv-123", "user-123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "unauthorized") {
		t.Errorf("Expected error message to contain 'unauthorized', got: %s", err.Error())
	}
}

func TestDeleteConversation_DatabaseError(t *testing.T) {
	now := time.Now()

	mockDB := &MockDatabase{
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
		DeleteConversationFunc: func(id string) error {
			return errors.New("database error")
		},
	}

	service := NewConversationService(mockDB)
	err := service.DeleteConversation("conv-123", "user-123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "failed to delete conversation") {
		t.Errorf("Expected error message to contain 'failed to delete conversation', got: %s", err.Error())
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
