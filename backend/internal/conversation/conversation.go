package conversation

import (
	"chat-app/internal/llm"
	"log"
	"sync"
)

// ConversationSession manages conversation history for a user session
type ConversationSession struct {
	ID       string           // Unique session ID
	Username string           // Username for this session
	Messages []llm.Message    // Conversation history
	mu       sync.RWMutex    // Thread-safe access
}

// SessionManager manages multiple conversation sessions
type SessionManager struct {
	sessions map[string]*ConversationSession
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*ConversationSession),
	}
}

// GetOrCreateSession gets or creates a conversation session
func (sm *SessionManager) GetOrCreateSession(sessionID, username string) *ConversationSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		return session
	}

	session := &ConversationSession{
		ID:       sessionID,
		Username: username,
		Messages: []llm.Message{},
	}
	sm.sessions[sessionID] = session
	log.Printf("[SESSION] Created new session %s for user %s", sessionID, username)
	return session
}

// GetSession retrieves an existing conversation session
func (sm *SessionManager) GetSession(sessionID string) *ConversationSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.sessions[sessionID]
}

// DeleteSession removes a conversation session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[sessionID]; exists {
		delete(sm.sessions, sessionID)
		log.Printf("[SESSION] Deleted session %s", sessionID)
	}
}

// AddUserMessage adds a user message to the conversation history
func (cs *ConversationSession) AddUserMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.Messages = append(cs.Messages, llm.Message{
		Role:    "user",
		Content: content,
	})
	log.Printf("[SESSION %s] Added user message, total messages: %d", cs.ID, len(cs.Messages))
}

// AddAssistantMessage adds an assistant message to the conversation history
func (cs *ConversationSession) AddAssistantMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.Messages = append(cs.Messages, llm.Message{
		Role:    "assistant",
		Content: content,
	})
	log.Printf("[SESSION %s] Added assistant message, total messages: %d", cs.ID, len(cs.Messages))
}

// GetMessages returns a copy of the conversation messages
func (cs *ConversationSession) GetMessages() []llm.Message {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Return a copy to prevent external modification
	messagesCopy := make([]llm.Message, len(cs.Messages))
	copy(messagesCopy, cs.Messages)
	return messagesCopy
}

// ClearHistory clears the conversation history
func (cs *ConversationSession) ClearHistory() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.Messages = []llm.Message{}
	log.Printf("[SESSION %s] Cleared conversation history", cs.ID)
}
