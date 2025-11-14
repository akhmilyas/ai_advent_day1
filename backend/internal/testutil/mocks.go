package testutil

import (
	"chat-app/internal/app"
	"chat-app/internal/config"
	"chat-app/internal/repository/db"
	"chat-app/internal/service/llm"
	"errors"
)

// MockDatabase is a mock implementation of db.Database for testing
type MockDatabase struct {
	// User mocks
	GetUserByUsernameFunc func(username string) (*db.User, error)
	CreateUserFunc        func(username, email, passwordHash string) (*db.User, error)

	// Conversation mocks
	GetConversationFunc        func(id string) (*db.Conversation, error)
	CreateConversationFunc     func(userID, title, format, schema string) (*db.Conversation, error)
	GetConversationsByUserFunc func(userID string) ([]db.Conversation, error)
	DeleteConversationFunc     func(id string) error

	// Message mocks
	AddMessageFunc                         func(conversationID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error)
	GetConversationMessagesFunc            func(conversationID string) ([]llm.Message, error)
	GetConversationMessagesWithDetailsFunc func(conversationID string) ([]db.Message, error)
	GetMessagesAfterMessageFunc            func(conversationID, afterMessageID string) ([]llm.Message, error)
	GetLastMessageIDFunc                   func(conversationID string) (*string, error)

	// Summary mocks
	GetActiveSummaryFunc               func(conversationID string) (*db.ConversationSummary, error)
	CreateSummaryFunc                  func(conversationID, summaryContent string, summarizedUpToMessageID *string) (*db.ConversationSummary, error)
	GetAllSummariesFunc                func(conversationID string) ([]db.ConversationSummary, error)
	IncrementSummaryUsageCountFunc     func(summaryID string) error
	UpdateConversationActiveSummaryFunc func(conversationID, summaryID string) error
}

// User methods
func (m *MockDatabase) GetUserByUsername(username string) (*db.User, error) {
	if m.GetUserByUsernameFunc != nil {
		return m.GetUserByUsernameFunc(username)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) CreateUser(username, email, passwordHash string) (*db.User, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(username, email, passwordHash)
	}
	return nil, errors.New("not implemented")
}

// Conversation methods
func (m *MockDatabase) GetConversation(id string) (*db.Conversation, error) {
	if m.GetConversationFunc != nil {
		return m.GetConversationFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) CreateConversation(userID, title, format, schema string) (*db.Conversation, error) {
	if m.CreateConversationFunc != nil {
		return m.CreateConversationFunc(userID, title, format, schema)
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

// Message methods
func (m *MockDatabase) AddMessage(conversationID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
	if m.AddMessageFunc != nil {
		return m.AddMessageFunc(conversationID, role, content, model, temperature, provider, generationID, promptTokens, completionTokens, totalTokens, totalCost, latency, generationTime)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetConversationMessages(conversationID string) ([]llm.Message, error) {
	if m.GetConversationMessagesFunc != nil {
		return m.GetConversationMessagesFunc(conversationID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetConversationMessagesWithDetails(conversationID string) ([]db.Message, error) {
	if m.GetConversationMessagesWithDetailsFunc != nil {
		return m.GetConversationMessagesWithDetailsFunc(conversationID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetMessagesAfterMessage(conversationID, afterMessageID string) ([]llm.Message, error) {
	if m.GetMessagesAfterMessageFunc != nil {
		return m.GetMessagesAfterMessageFunc(conversationID, afterMessageID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetLastMessageID(conversationID string) (*string, error) {
	if m.GetLastMessageIDFunc != nil {
		return m.GetLastMessageIDFunc(conversationID)
	}
	return nil, errors.New("not implemented")
}

// Summary methods
func (m *MockDatabase) GetActiveSummary(conversationID string) (*db.ConversationSummary, error) {
	if m.GetActiveSummaryFunc != nil {
		return m.GetActiveSummaryFunc(conversationID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) CreateSummary(conversationID, summaryContent string, summarizedUpToMessageID *string) (*db.ConversationSummary, error) {
	if m.CreateSummaryFunc != nil {
		return m.CreateSummaryFunc(conversationID, summaryContent, summarizedUpToMessageID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) GetAllSummaries(conversationID string) ([]db.ConversationSummary, error) {
	if m.GetAllSummariesFunc != nil {
		return m.GetAllSummariesFunc(conversationID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDatabase) IncrementSummaryUsageCount(summaryID string) error {
	if m.IncrementSummaryUsageCountFunc != nil {
		return m.IncrementSummaryUsageCountFunc(summaryID)
	}
	return errors.New("not implemented")
}

func (m *MockDatabase) UpdateConversationActiveSummary(conversationID, summaryID string) error {
	if m.UpdateConversationActiveSummaryFunc != nil {
		return m.UpdateConversationActiveSummaryFunc(conversationID, summaryID)
	}
	return errors.New("not implemented")
}

// MockLLMProvider is a mock implementation of llm.LLMProvider for testing
type MockLLMProvider struct {
	ChatWithHistoryFunc       func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error)
	ChatWithHistoryStreamFunc func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan llm.StreamChunk, error)
	ChatForSummarizationFunc  func(messages []llm.Message, summarizationPrompt string, modelOverride string, temperature *float64) (string, error)
	FetchGenerationCostFunc   func(generationID string) (*llm.GenerationData, error)
	GetDefaultModelFunc       func() string
}

func (m *MockLLMProvider) ChatWithHistory(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
	if m.ChatWithHistoryFunc != nil {
		return m.ChatWithHistoryFunc(messages, customSystemPrompt, format, modelOverride, temperature)
	}
	return "", errors.New("not implemented")
}

func (m *MockLLMProvider) ChatWithHistoryStream(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan llm.StreamChunk, error) {
	if m.ChatWithHistoryStreamFunc != nil {
		return m.ChatWithHistoryStreamFunc(messages, customSystemPrompt, format, modelOverride, temperature)
	}
	return nil, errors.New("not implemented")
}

func (m *MockLLMProvider) ChatForSummarization(messages []llm.Message, summarizationPrompt string, modelOverride string, temperature *float64) (string, error) {
	if m.ChatForSummarizationFunc != nil {
		return m.ChatForSummarizationFunc(messages, summarizationPrompt, modelOverride, temperature)
	}
	return "", errors.New("not implemented")
}

func (m *MockLLMProvider) FetchGenerationCost(generationID string) (*llm.GenerationData, error) {
	if m.FetchGenerationCostFunc != nil {
		return m.FetchGenerationCostFunc(generationID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockLLMProvider) GetDefaultModel() string {
	if m.GetDefaultModelFunc != nil {
		return m.GetDefaultModelFunc()
	}
	return "default-model"
}

// NewMockConfig creates a mock app.Config for testing
func NewMockConfig() *app.Config {
	// Create ModelsConfig (models field is unexported, so config will have empty models list)
	modelsConfig := &config.ModelsConfig{}

	return &app.Config{
		AppConfig: &config.AppConfig{
			LLM: config.LLMConfig{
				OpenRouterAPIKey:    "test-api-key",
				DefaultSystemPrompt: "You are a helpful assistant.",
				SummarizationPrompt: "Summarize the conversation.",
				TextTopP:            0.9,
				TextTopK:            40,
				StructuredTopP:      0.8,
				StructuredTopK:      20,
			},
			Models: modelsConfig,
		},
	}
}

// NewMockModelsConfig creates a ModelsConfig with test models
// Note: This uses internal knowledge that models is an unexported field
// In a real scenario, you'd load from a test JSON file
func NewMockModelsConfig() *config.ModelsConfig {
	// Since models field is unexported, we can't create it directly in tests
	// The best approach is to use a test JSON file, but for simplicity
	// we return a nil-safe config
	return &config.ModelsConfig{}
}
