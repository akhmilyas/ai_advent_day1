1. Security Issues (HIGH PRIORITY)

  auth/auth.go:19 - Hardcoded JWT Secret

  var jwtSecret = []byte("your-secret-key-change-in-production")
  Issue: Secret key is hardcoded in source code
  Impact: Security vulnerability - anyone with source code access can forge tokens
  Fix:
  - Move to environment variable
  - Use a strong, random key (at least 32 bytes)
  - Add validation on startup to ensure secret is configured

  func getJWTSecret() []byte {
      secret := os.Getenv("JWT_SECRET")
      if secret == "" {
          log.Fatal("JWT_SECRET environment variable must be set")
      }
      if len(secret) < 32 {
          log.Fatal("JWT_SECRET must be at least 32 characters")
      }
      return []byte(secret)
  }

  ---
  2. Error Handling & Response Consistency

  handlers/chat.go - Inconsistent Error Responses

  Issue: Some errors use http.Error(), others use JSON responses
  Example:
  // Line 204: JSON response
  json.NewEncoder(w).Encode(ChatResponse{Error: err.Error()})

  // Line 205: Plain text response
  http.Error(w, "Invalid model specified", http.StatusBadRequest)

  Fix: Create a standardized error response helper:
  type ErrorResponse struct {
      Error   string `json:"error"`
      Code    int    `json:"code"`
      Message string `json:"message"`
  }

  func (ch *ChatHandlers) sendError(w http.ResponseWriter, status int, message string, err error) {
      w.Header().Set("Content-Type", "application/json")
      w.WriteHeader(status)

      errResp := ErrorResponse{
          Code:    status,
          Message: message,
      }
      if err != nil {
          errResp.Error = err.Error()
      }
      json.NewEncoder(w).Encode(errResp)
  }

  ---
  3. Database Layer - Missing Interface

  db/ - Direct Database Calls Throughout

  Issue: Handlers call db.GetConversation() directly - no abstraction layer
  Impact:
  - Hard to test (cannot mock database)
  - Tight coupling between layers
  - Difficult to swap database implementations

  Fix: Create database interface:
  // internal/db/interface.go
  type Database interface {
      // Users
      GetUserByUsername(username string) (*User, error)
      CreateUser(username, email, password string) (*User, error)

      // Conversations
      GetConversation(id string) (*Conversation, error)
      CreateConversation(userID, title, format, schema string) (*Conversation, error)
      GetConversationsByUser(userID string) ([]Conversation, error)
      DeleteConversation(id string) error

      // Messages
      AddMessage(convID, role, content, model string, temp *float64, ...) (*Message, error)
      GetConversationMessages(convID string) ([]llm.Message, error)

      // Summaries
      GetActiveSummary(convID string) (*ConversationSummary, error)
      CreateSummary(convID, content string, upToMsgID *string) (*ConversationSummary, error)
  }

  // PostgresDB implements Database
  type PostgresDB struct {
      db *sql.DB
  }

  Then inject into config:
  type Config struct {
      DB           db.Database
      ModelsConfig *config.ModelsConfig
  }

  ---
  4. Configuration Management

  Multiple Config Sources

  Issue: Configuration scattered across:
  - Environment variables (os.Getenv() calls throughout)
  - JSON files (models.json)
  - Hardcoded values
  - Global variables

  Fix: Centralized configuration struct:
  // internal/config/app.go
  type AppConfig struct {
      Server    ServerConfig
      Database  DatabaseConfig
      LLM       LLMConfig
      Auth      AuthConfig
      Models    *ModelsConfig
  }

  type ServerConfig struct {
      Port string
  }

  type DatabaseConfig struct {
      Host     string
      Port     string
      User     string
      Password string
      Name     string
      SSLMode  string
  }

  type LLMConfig struct {
      OpenRouterAPIKey          string
      DefaultSystemPrompt       string
      TextTopP                  float64
      TextTopK                  int
      StructuredTopP            float64
      StructuredTopK            int
      SummarizationPrompt       string
  }

  type AuthConfig struct {
      JWTSecret        []byte
      TokenExpiration  time.Duration
  }

  func LoadConfig() (*AppConfig, error) {
      // Load from environment, validate, set defaults
  }

  ---
  5. Logging Improvements

  Inconsistent Logging

  Issue: Mix of log.Printf, fmt.Println, manual string formatting
  Example:
  log.Printf("[CHAT] User input: %s", req.Message)  // Structured
  fmt.Println("Warning: OPENROUTER_API_KEY environment variable not set")  // Unstructured

  Fix: Use structured logging library:
  import "github.com/sirupsen/logrus"

  // Global logger
  var log = logrus.New()

  // Usage
  log.WithFields(logrus.Fields{
      "user":           username,
      "conversation":   convID,
      "message_length": len(req.Message),
  }).Info("Processing chat request")

  ---
  6. Handler Structure - Code Duplication

  SummarizeConversationHandler - Complex Conditional Logic

  Lines 755-809: Complex branching logic for summary creation vs reuse
  Fix: Extract to separate methods:
  func (ch *ChatHandlers) shouldCreateNewSummary(summary *db.ConversationSummary) bool {
      return summary == nil || summary.UsageCount >= 2
  }

  func (ch *ChatHandlers) buildSummarizationInput(convID string, summary *db.ConversationSummary) ([]llm.Message, *string, error) {
      if summary == nil {
          return ch.getAllMessagesForSummary(convID)
      }
      return ch.getIncrementalSummaryInput(convID, summary)
  }

  ---
  7. Validation Layer Missing

  Request Validation

  Issue: Validation scattered in handlers
  Example:
  if req.Message == "" {
      http.Error(w, "Message cannot be empty", http.StatusBadRequest)
      return
  }

  Fix: Create validation layer:
  // internal/validation/chat.go
  type ChatRequestValidator struct{}

  func (v *ChatRequestValidator) Validate(req *ChatRequest) error {
      if req.Message == "" {
          return errors.New("message cannot be empty")
      }
      if req.WarAndPeacePercent < 0 || req.WarAndPeacePercent > 100 {
          return errors.New("war_and_peace_percent must be between 0 and 100")
      }
      if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
          return errors.New("temperature must be between 0 and 2")
      }
      return nil
  }

  ---
  8. Context Timeout Missing

  HTTP Handlers

  Issue: No timeouts on long-running operations (LLM calls, database queries)
  Impact: Goroutine leaks, resource exhaustion

  Fix:
  func (ch *ChatHandlers) ChatStreamHandler(w http.ResponseWriter, r *http.Request) {
      // Create context with timeout
      ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
      defer cancel()

      // Pass context to LLM calls
      chunks, err := provider.ChatWithHistoryStreamContext(ctx, ...)
  }

  ---
  9. Global State & Singleton Pattern

  db/postgres.go:13-15

  var (
      instance *sql.DB
      once     sync.Once
  )
  Issue: Global singleton makes testing difficult
  Fix: Factory pattern with proper initialization:
  type DB struct {
      conn *sql.DB
  }

  func NewDB(config DatabaseConfig) (*DB, error) {
      dsn := buildDSN(config)
      conn, err := sql.Open("postgres", dsn)
      if err != nil {
          return nil, err
      }
      return &DB{conn: conn}, nil
  }

  ---
  10. Missing Middleware for Common Concerns

  Suggested Middleware:
  // Logging middleware
  func LoggingMiddleware(next http.Handler) http.Handler {
      return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
          start := time.Now()
          next.ServeHTTP(w, r)
          log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
      })
  }

  // Recovery middleware (panic recovery)
  func RecoveryMiddleware(next http.Handler) http.Handler {
      return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
          defer func() {
              if err := recover(); err != nil {
                  log.Printf("Panic recovered: %v", err)
                  http.Error(w, "Internal server error", http.StatusInternalServerError)
              }
          }()
          next.ServeHTTP(w, r)
      })
  }

  // Rate limiting middleware
  func RateLimitMiddleware(next http.Handler) http.Handler {
      // Use golang.org/x/time/rate
  }

  ---
  11. Testing Infrastructure

  Missing:
  - Unit tests for handlers
  - Integration tests for database
  - Mock implementations for LLM provider
  - Table-driven tests

  Example test structure:
  // handlers/chat_test.go
  func TestChatHandler_ValidRequest(t *testing.T) {
      mockDB := &mockDatabase{}
      mockLLM := &mockLLMProvider{}
      config := &app.Config{DB: mockDB, ...}

      handler := NewChatHandlers(config)

      req := httptest.NewRequest("POST", "/api/chat", body)
      rec := httptest.NewRecorder()

      handler.ChatHandler(rec, req)

      assert.Equal(t, http.StatusOK, rec.Code)
  }

  ---
  12. Environment-Specific Configuration

  Issue: No distinction between dev/staging/production configs
  Fix: Environment-based config files:
  config/
    ├── development.yaml
    ├── staging.yaml
    └── production.yaml

  ---
  13. Code Organization Improvements

  Suggested Structure:
  backend/
  ├── cmd/
  │   └── server/
  │       └── main.go (thin, just wiring)
  ├── internal/
  │   ├── api/          (HTTP layer)
  │   │   ├── handlers/
  │   │   ├── middleware/
  │   │   └── router/
  │   ├── domain/       (Business logic)
  │   │   ├── conversation/
  │   │   ├── message/
  │   │   └── user/
  │   ├── repository/   (Data access)
  │   │   └── postgres/
  │   ├── service/      (Application services)
  │   │   ├── llm/
  │   │   └── summary/
  │   └── config/
  └── pkg/              (Reusable packages)
      ├── validation/
      └── errors/

  ---
  Priority Recommendations

  HIGH (Security & Critical):
  1. ✅ Move JWT secret to environment variable
  2. ✅ Add request validation layer
  3. ✅ Implement standardized error responses
  4. ✅ Add context timeouts to prevent resource leaks

  MEDIUM (Maintainability):
  5. ✅ Create database interface for testability
  6. ✅ Centralize configuration management
  7. ✅ Add structured logging
  8. ✅ Extract complex conditional logic

  LOW (Nice to Have):
  9. Add comprehensive test coverage
  10. Implement rate limiting
  11. Add health check endpoints with detailed status
  12. Add metrics/monitoring (Prometheus)

