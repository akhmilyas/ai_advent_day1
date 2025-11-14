package chat

import (
	"chat-app/internal/app"
	"chat-app/internal/config"
	"chat-app/internal/repository/db"
	"chat-app/internal/service/llm"
	"chat-app/internal/testutil"
	"errors"
	"testing"
)

// Test NewChatService
func TestNewChatService(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockConfig := testutil.NewMockConfig()

	service := NewChatService(mockDB, mockConfig)

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if service.db == nil {
		t.Error("Expected db to be set")
	}

	if service.config == nil {
		t.Error("Expected config to be set")
	}

	if service.llmProvider == nil {
		t.Error("Expected llmProvider to be set")
	}
}

// Test SendMessage - Success scenarios
func TestSendMessage_Success(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	conversationID := "conv-123"
	userID := "user-456"
	message := "Hello, world!"
	expectedResponse := "Hi there!"

	// Mock conversation retrieval
	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:             conversationID,
			UserID:         userID,
			Title:          "Test Conversation",
			ResponseFormat: "text",
			ResponseSchema: "",
		}, nil
	}

	// Mock message saving
	messageCount := 0
	mockDB.AddMessageFunc = func(convID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
		messageCount++
		return &db.Message{
			ID:             "msg-" + string(rune(messageCount)),
			ConversationID: convID,
			Role:           role,
			Content:        content,
		}, nil
	}

	// Mock conversation history
	mockDB.GetActiveSummaryFunc = func(convID string) (*db.ConversationSummary, error) {
		return nil, errors.New("no summary")
	}

	mockDB.GetConversationMessagesFunc = func(convID string) ([]llm.Message, error) {
		return []llm.Message{
			{Role: "user", Content: "Previous message"},
			{Role: "assistant", Content: "Previous response"},
		}, nil
	}

	// Mock LLM response
	mockLLM.ChatWithHistoryFunc = func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
		return expectedResponse, nil
	}

	mockLLM.GetDefaultModelFunc = func() string {
		return "test-model"
	}

	// Execute test
	req := SendMessageRequest{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
		SystemPrompt:   "You are helpful",
		ResponseFormat: "text",
	}

	response, err := service.SendMessage(req)

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response.Response)
	}

	if response.ConversationID != conversationID {
		t.Errorf("Expected conversation ID '%s', got '%s'", conversationID, response.ConversationID)
	}

	if response.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", response.Model)
	}

	if messageCount != 2 {
		t.Errorf("Expected 2 messages to be saved (user + assistant), got %d", messageCount)
	}
}

// Test SendMessage - Create new conversation
func TestSendMessage_CreateNewConversation(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	userID := "user-456"
	message := "This is a very long message that exceeds 100 characters and should be truncated when used as the conversation title. Lorem ipsum dolor sit amet, consectetur adipiscing elit."
	// Title should be truncated to 100 runes (not bytes)
	runes := []rune(message)
	expectedTitle := string(runes[:100])

	// Mock conversation creation
	var createdTitle string
	mockDB.CreateConversationFunc = func(uid, title, format, schema string) (*db.Conversation, error) {
		createdTitle = title
		return &db.Conversation{
			ID:             "new-conv-123",
			UserID:         uid,
			Title:          title,
			ResponseFormat: format,
			ResponseSchema: schema,
		}, nil
	}

	// Mock message saving
	mockDB.AddMessageFunc = func(convID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
		return &db.Message{ID: "msg-1", ConversationID: convID, Role: role, Content: content}, nil
	}

	// Mock conversation history
	mockDB.GetActiveSummaryFunc = func(convID string) (*db.ConversationSummary, error) {
		return nil, errors.New("no summary")
	}

	mockDB.GetConversationMessagesFunc = func(convID string) ([]llm.Message, error) {
		return []llm.Message{}, nil
	}

	// Mock LLM response
	mockLLM.ChatWithHistoryFunc = func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
		return "Response", nil
	}

	mockLLM.GetDefaultModelFunc = func() string {
		return "test-model"
	}

	// Execute test (no conversation ID provided)
	req := SendMessageRequest{
		Message:        message,
		ConversationID: "", // Empty to trigger creation
		UserID:         userID,
		SystemPrompt:   "You are helpful",
		ResponseFormat: "text",
	}

	response, err := service.SendMessage(req)

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.ConversationID != "new-conv-123" {
		t.Errorf("Expected new conversation ID 'new-conv-123', got '%s'", response.ConversationID)
	}

	if createdTitle != expectedTitle {
		t.Errorf("Expected title to be truncated to 100 chars.\nExpected: '%s'\nGot:      '%s'", expectedTitle, createdTitle)
	}
}

// Test SendMessage - Unauthorized access
func TestSendMessage_Unauthorized(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	// Mock conversation with different owner
	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:     "conv-123",
			UserID: "other-user", // Different user
		}, nil
	}

	req := SendMessageRequest{
		Message:        "Hello",
		ConversationID: "conv-123",
		UserID:         "user-456", // Requesting user
	}

	_, err := service.SendMessage(req)

	if err == nil {
		t.Fatal("Expected unauthorized error, got nil")
	}

	if err.Error() != "unauthorized: user does not own this conversation" {
		t.Errorf("Expected unauthorized error, got: %v", err)
	}
}

// Test SendMessage - Invalid model
func TestSendMessage_InvalidModel(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}

	// Create config with actual models file for proper validation
	modelsConfig, err := config.NewModelsConfig("../../../config/models.json")
	if err != nil {
		t.Fatalf("Failed to load models config: %v", err)
	}

	mockConfig := &app.Config{
		AppConfig: &config.AppConfig{
			Models: modelsConfig,
		},
	}

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:             "conv-123",
			UserID:         "user-456",
			ResponseFormat: "text",
		}, nil
	}

	req := SendMessageRequest{
		Message:        "Hello",
		ConversationID: "conv-123",
		UserID:         "user-456",
		Model:          "invalid-model-xyz", // Invalid model
	}

	_, err = service.SendMessage(req)

	if err == nil {
		t.Fatal("Expected invalid model error, got nil")
	}

	if err.Error() != "invalid model: invalid model specified" {
		t.Errorf("Expected invalid model error, got: %v", err)
	}
}

// Test SendMessage - Database error saving user message
func TestSendMessage_DatabaseErrorSavingUserMessage(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:             "conv-123",
			UserID:         "user-456",
			ResponseFormat: "text",
		}, nil
	}

	mockDB.AddMessageFunc = func(convID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
		return nil, errors.New("database error")
	}

	req := SendMessageRequest{
		Message:        "Hello",
		ConversationID: "conv-123",
		UserID:         "user-456",
	}

	_, err := service.SendMessage(req)

	if err == nil {
		t.Fatal("Expected database error, got nil")
	}

	if err.Error() != "failed to save user message: database error" {
		t.Errorf("Expected database error, got: %v", err)
	}
}

// Test SendMessage - LLM error
func TestSendMessage_LLMError(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:             "conv-123",
			UserID:         "user-456",
			ResponseFormat: "text",
		}, nil
	}

	mockDB.AddMessageFunc = func(convID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
		return &db.Message{ID: "msg-1"}, nil
	}

	mockDB.GetActiveSummaryFunc = func(convID string) (*db.ConversationSummary, error) {
		return nil, errors.New("no summary")
	}

	mockDB.GetConversationMessagesFunc = func(convID string) ([]llm.Message, error) {
		return []llm.Message{}, nil
	}

	mockLLM.ChatWithHistoryFunc = func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
		return "", errors.New("LLM API error")
	}

	mockLLM.GetDefaultModelFunc = func() string {
		return "test-model"
	}

	req := SendMessageRequest{
		Message:        "Hello",
		ConversationID: "conv-123",
		UserID:         "user-456",
	}

	_, err := service.SendMessage(req)

	if err == nil {
		t.Fatal("Expected LLM error, got nil")
	}

	if err.Error() != "LLM error: LLM API error" {
		t.Errorf("Expected LLM error, got: %v", err)
	}
}

// Test SendMessage - With active summary
func TestSendMessage_WithActiveSummary(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	summaryID := "summary-789"
	lastMessageID := "msg-100"

	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:             "conv-123",
			UserID:         "user-456",
			ResponseFormat: "text",
		}, nil
	}

	mockDB.AddMessageFunc = func(convID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
		return &db.Message{ID: "msg-new", ConversationID: convID, Role: role, Content: content}, nil
	}

	// Mock active summary
	mockDB.GetActiveSummaryFunc = func(convID string) (*db.ConversationSummary, error) {
		return &db.ConversationSummary{
			ID:                      summaryID,
			ConversationID:          convID,
			SummaryContent:          "This is a summary of previous messages",
			SummarizedUpToMessageID: &lastMessageID,
			UsageCount:              1,
		}, nil
	}

	// Mock messages after summary
	mockDB.GetMessagesAfterMessageFunc = func(convID, afterMsgID string) ([]llm.Message, error) {
		if afterMsgID != lastMessageID {
			t.Errorf("Expected to get messages after '%s', got '%s'", lastMessageID, afterMsgID)
		}
		return []llm.Message{
			{Role: "user", Content: "Recent message"},
			{Role: "assistant", Content: "Recent response"},
		}, nil
	}

	// Mock increment summary usage
	usageIncremented := false
	mockDB.IncrementSummaryUsageCountFunc = func(sid string) error {
		if sid != summaryID {
			t.Errorf("Expected to increment summary '%s', got '%s'", summaryID, sid)
		}
		usageIncremented = true
		return nil
	}

	// Mock LLM call to verify system prompt contains summary
	var receivedSystemPrompt string
	mockLLM.ChatWithHistoryFunc = func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
		receivedSystemPrompt = customSystemPrompt
		return "Response with summary context", nil
	}

	mockLLM.GetDefaultModelFunc = func() string {
		return "test-model"
	}

	req := SendMessageRequest{
		Message:        "Hello",
		ConversationID: "conv-123",
		UserID:         "user-456",
		SystemPrompt:   "You are helpful",
	}

	response, err := service.SendMessage(req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Response != "Response with summary context" {
		t.Errorf("Expected response with summary context, got: %s", response.Response)
	}

	if !usageIncremented {
		t.Error("Expected summary usage count to be incremented")
	}

	// Verify system prompt contains summary
	expectedSummaryInPrompt := "Previous conversation summary:\nThis is a summary of previous messages"
	if len(receivedSystemPrompt) < len(expectedSummaryInPrompt) || receivedSystemPrompt[:len(expectedSummaryInPrompt)] != expectedSummaryInPrompt {
		t.Errorf("Expected system prompt to start with summary context.\nGot: %s", receivedSystemPrompt)
	}
}

// Test SendMessage - JSON format with schema
func TestSendMessage_JSONFormatWithSchema(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	schema := `{"type": "object", "properties": {"name": {"type": "string"}}}`

	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:             "conv-123",
			UserID:         "user-456",
			ResponseFormat: "json",
			ResponseSchema: schema,
		}, nil
	}

	mockDB.AddMessageFunc = func(convID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
		return &db.Message{ID: "msg-1", ConversationID: convID, Role: role, Content: content}, nil
	}

	mockDB.GetActiveSummaryFunc = func(convID string) (*db.ConversationSummary, error) {
		return nil, errors.New("no summary")
	}

	mockDB.GetConversationMessagesFunc = func(convID string) ([]llm.Message, error) {
		return []llm.Message{}, nil
	}

	var receivedSystemPrompt string
	mockLLM.ChatWithHistoryFunc = func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (string, error) {
		receivedSystemPrompt = customSystemPrompt
		return `{"name": "Test"}`, nil
	}

	mockLLM.GetDefaultModelFunc = func() string {
		return "test-model"
	}

	req := SendMessageRequest{
		Message:        "Create JSON",
		ConversationID: "conv-123",
		UserID:         "user-456",
		SystemPrompt:   "Custom prompt", // Should be ignored for JSON format
	}

	_, err := service.SendMessage(req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify system prompt contains JSON schema instructions
	if receivedSystemPrompt == "" {
		t.Error("Expected system prompt to be set")
	}

	// Should contain schema validation instructions
	expectedSchemaInPrompt := schema
	if len(receivedSystemPrompt) == 0 || !contains(receivedSystemPrompt, expectedSchemaInPrompt) {
		t.Errorf("Expected system prompt to contain schema.\nGot: %s", receivedSystemPrompt)
	}

	// Should contain JSON-specific instructions
	if !contains(receivedSystemPrompt, "valid JSON") {
		t.Errorf("Expected system prompt to contain JSON instructions.\nGot: %s", receivedSystemPrompt)
	}
}

// Test SendMessageStream - Success
func TestSendMessageStream_Success(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	conversationID := "conv-123"
	userID := "user-456"

	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:             conversationID,
			UserID:         userID,
			ResponseFormat: "text",
		}, nil
	}

	messagesSaved := []string{}
	mockDB.AddMessageFunc = func(convID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
		messagesSaved = append(messagesSaved, role+":"+content)
		return &db.Message{ID: "msg-" + role, ConversationID: convID, Role: role, Content: content}, nil
	}

	mockDB.GetActiveSummaryFunc = func(convID string) (*db.ConversationSummary, error) {
		return nil, errors.New("no summary")
	}

	mockDB.GetConversationMessagesFunc = func(convID string) ([]llm.Message, error) {
		return []llm.Message{}, nil
	}

	// Mock streaming LLM response
	mockLLM.ChatWithHistoryStreamFunc = func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan llm.StreamChunk, error) {
		ch := make(chan llm.StreamChunk)
		go func() {
			defer close(ch)
			// Send content chunks
			ch <- llm.StreamChunk{Content: "Hello"}
			ch <- llm.StreamChunk{Content: " world"}
			ch <- llm.StreamChunk{Content: "!"}
			// Send metadata
			ch <- llm.StreamChunk{
				Metadata: &llm.StreamMetadata{
					GenerationID: "gen-123",
					Usage: &llm.ResponseUsage{
						PromptTokens:     10,
						CompletionTokens: 5,
						TotalTokens:      15,
					},
				},
			}
		}()
		return ch, nil
	}

	mockLLM.GetDefaultModelFunc = func() string {
		return "test-model"
	}

	mockLLM.FetchGenerationCostFunc = func(generationID string) (*llm.GenerationData, error) {
		return &llm.GenerationData{
			NativeTokensPrompt:     10,
			NativeTokensCompletion: 5,
			TotalCost:              0.001,
			Latency:                100,
			GenerationTime:         500,
		}, nil
	}

	req := SendMessageRequest{
		Message:        "Hello",
		ConversationID: conversationID,
		UserID:         userID,
	}

	chunks, err := service.SendMessageStream(req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Collect chunks
	var receivedContent string
	var receivedConvID string
	var receivedModel string
	var receivedMetadata *llm.StreamMetadata

	for chunk := range chunks {
		if chunk.Content != "" {
			receivedContent += chunk.Content
		}
		if chunk.ConvID != "" {
			receivedConvID = chunk.ConvID
		}
		if chunk.Model != "" {
			receivedModel = chunk.Model
		}
		if chunk.Metadata != nil {
			receivedMetadata = chunk.Metadata
		}
	}

	// Verify content
	expectedContent := "Hello world!"
	if receivedContent != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, receivedContent)
	}

	// Verify metadata
	if receivedConvID != conversationID {
		t.Errorf("Expected conversation ID '%s', got '%s'", conversationID, receivedConvID)
	}

	if receivedModel != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", receivedModel)
	}

	if receivedMetadata == nil {
		t.Fatal("Expected metadata to be received")
	}

	if receivedMetadata.GenerationID != "gen-123" {
		t.Errorf("Expected generation ID 'gen-123', got '%s'", receivedMetadata.GenerationID)
	}

	// Verify messages saved (user + assistant)
	if len(messagesSaved) != 2 {
		t.Errorf("Expected 2 messages saved, got %d", len(messagesSaved))
	}

	if messagesSaved[0] != "user:Hello" {
		t.Errorf("Expected first message 'user:Hello', got '%s'", messagesSaved[0])
	}

	if messagesSaved[1] != "assistant:Hello world!" {
		t.Errorf("Expected second message 'assistant:Hello world!', got '%s'", messagesSaved[1])
	}
}

// Test SendMessageStream - Unauthorized
func TestSendMessageStream_Unauthorized(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:     "conv-123",
			UserID: "other-user",
		}, nil
	}

	req := SendMessageRequest{
		Message:        "Hello",
		ConversationID: "conv-123",
		UserID:         "user-456",
	}

	_, err := service.SendMessageStream(req)

	if err == nil {
		t.Fatal("Expected unauthorized error, got nil")
	}

	if err.Error() != "unauthorized: user does not own this conversation" {
		t.Errorf("Expected unauthorized error, got: %v", err)
	}
}

// Test SendMessageStream - LLM streaming error
func TestSendMessageStream_LLMStreamingError(t *testing.T) {
	mockDB := &testutil.MockDatabase{}
	mockLLM := &testutil.MockLLMProvider{}
	mockConfig := createMockConfigWithValidation()

	service := &ChatService{
		db:          mockDB,
		config:      mockConfig,
		llmProvider: mockLLM,
	}

	mockDB.GetConversationFunc = func(id string) (*db.Conversation, error) {
		return &db.Conversation{
			ID:             "conv-123",
			UserID:         "user-456",
			ResponseFormat: "text",
		}, nil
	}

	mockDB.AddMessageFunc = func(convID, role, content, model string, temperature *float64, provider, generationID string, promptTokens, completionTokens, totalTokens *int, totalCost *float64, latency, generationTime *int) (*db.Message, error) {
		return &db.Message{ID: "msg-1"}, nil
	}

	mockDB.GetActiveSummaryFunc = func(convID string) (*db.ConversationSummary, error) {
		return nil, errors.New("no summary")
	}

	mockDB.GetConversationMessagesFunc = func(convID string) ([]llm.Message, error) {
		return []llm.Message{}, nil
	}

	mockLLM.ChatWithHistoryStreamFunc = func(messages []llm.Message, customSystemPrompt string, format string, modelOverride string, temperature *float64) (<-chan llm.StreamChunk, error) {
		return nil, errors.New("streaming error")
	}

	mockLLM.GetDefaultModelFunc = func() string {
		return "test-model"
	}

	req := SendMessageRequest{
		Message:        "Hello",
		ConversationID: "conv-123",
		UserID:         "user-456",
	}

	_, err := service.SendMessageStream(req)

	if err == nil {
		t.Fatal("Expected streaming error, got nil")
	}

	if err.Error() != "LLM streaming error: streaming error" {
		t.Errorf("Expected streaming error, got: %v", err)
	}
}

// Helper function to create a mock config with model validation
func createMockConfigWithValidation() *app.Config {
	// Create a simple config that will pass model validation
	// Since ModelsConfig has unexported fields, we use a test JSON file approach
	cfg := testutil.NewMockConfig()
	return cfg
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
