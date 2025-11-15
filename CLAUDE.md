# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Commands

### Docker Compose (Recommended)
```bash
docker compose build      # Build both frontend and backend images
docker compose up         # Start all services (PostgreSQL, Backend, Frontend)
docker compose down       # Stop all services
docker compose down -v    # Stop and remove volumes (clean database)
```

### Manual Backend Build & Run
Requires **Go 1.25.3+** (uses Go 1.22+ HTTP routing features)

```bash
cd backend
go mod download
go build -o server ./cmd/server
./server                  # Requires PostgreSQL running on localhost:5432
```

### Manual Frontend Build & Run
Built with **Vite** (not Create React App)

```bash
cd frontend
npm install
npm run build             # Production build with Vite
npm run dev               # Development server with HMR (http://localhost:3000)
npm start                 # Alias for npm run dev
npm test                  # Run tests with Vitest
```

### Testing
```bash
# Frontend tests
cd frontend && npm test

# Backend tests
cd backend && go test ./...
cd backend && go test ./pkg/validation -v  # Single package with verbose output
```

### Database Migrations
```bash
# Check migration status
docker compose exec postgres psql -U postgres -d chatapp -c "SELECT * FROM schema_migrations;"

# Migrations auto-apply on backend startup via golang-migrate
# Manual migration commands (if needed):
cd backend
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable" up
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable" down
```

## Architecture Overview

### High-Level Design
This is a fullstack chat application with **clean layered architecture**:

1. **Frontend (React/TypeScript + Vite)** on port 3000
   - Built with **Vite** for fast development and optimized production builds
   - State management via **Zustand** with persist and devtools middleware
   - Single-page app with login/register and chat interface
   - Connects to backend via HTTP REST (auth, models) and Server-Sent Events (chat streaming)
   - State: chat messages, conversations, settings (model, temperature, prompts, formats)
   - Persistence: localStorage for auth tokens, preferences, and cached state

2. **Backend (Go) - Clean Architecture** on port 8080
   - **API Layer** (`internal/api/handlers/`): HTTP request/response, auth middleware, SSE streaming
   - **Service Layer** (`internal/service/`): Business logic for chat, conversations, summaries, LLM providers
   - **Repository Layer** (`internal/repository/`): Database interface abstraction with PostgreSQL implementation
   - **Validation Layer** (`pkg/validation/`): Request validators with comprehensive test coverage
   - **Config Layer** (`internal/config/`): Centralized configuration from environment + JSON files

3. **Database (PostgreSQL)** on port 5432
   - Managed via **golang-migrate** (7 migrations tracked)
   - Five tables: users, conversations, messages, conversation_summaries, schema_migrations
   - Cascading deletes for referential integrity
   - Indexes on foreign keys and frequently queried fields
   - Cost/performance tracking: prompt_tokens, completion_tokens, total_cost, latency, generation_time

### Layered Backend Architecture

```
HTTP Request
  ↓
API Handlers (internal/api/handlers/)
  - Authentication middleware
  - Request parsing and validation
  - Response formatting (JSON, SSE)
  ↓
Service Layer (internal/service/)
  - ChatService: Message streaming, history management
  - ConversationService: CRUD operations
  - SummaryService: Summarization logic
  - LLM Providers: OpenRouterProvider, GenkitProvider
  ↓
Repository Layer (internal/repository/)
  - Interface definitions (repository.go)
  - PostgreSQL implementation (postgres_repository.go)
  - Database connection management
  ↓
PostgreSQL Database
```

**Key Benefits:**
- **Testability**: Service and repository layers use interfaces for easy mocking
- **Separation of Concerns**: Each layer has a single responsibility
- **Maintainability**: Business logic isolated from HTTP and database details
- **Flexibility**: Can swap database implementation without changing service layer

### Data Flow for Chat (Streaming)

```
User sends message
  ↓
Frontend: ChatService.streamMessage()
  ↓
POST /api/chat/stream {
  message,
  conversation_id?,
  system_prompt?,
  response_format?,
  response_schema?,
  model?,
  temperature?,
  provider?,              // NEW: 'openrouter' or 'genkit'
  use_war_and_peace?,     // NEW: Enable War and Peace context injection
  war_and_peace_percent?  // NEW: Percentage of text to append (1-100)
}
  ↓
API Handler: ChatStreamHandler
  - Validates JWT token
  - Validates request via validation.ChatRequestValidator
  - Extracts user from context
  ↓
Service: ChatService.SendMessageStream()
  - Validates model ID against config
  - Gets/creates conversation (with format/schema if new)
  - Adds user message to repository
  - Fetches conversation history
  - Checks for active summary (if exists, use summary + recent messages)
  - Builds format-specific system prompt
  - Appends War and Peace text if requested
  ↓
LLM Provider (OpenRouterProvider or GenkitProvider)
  - Selects format-aware top_p/top_k from config
  - Uses user-provided temperature (0.0-2.0)
  - Builds message array: [system_prompt, user1, assistant1, ..., user_n]
  - Calls provider API with model, temperature, top_p, top_k
  - Streams response via SSE format (data: {chunk}\n\n)
  - Sends metadata: MODEL:, TEMPERATURE:, CONV_ID:, USAGE:
  ↓
Service: Saves message to repository
  - Records model, temperature, provider used
  - Fetches cost/token data from OpenRouter API (async with retry)
  - Stores prompt_tokens, completion_tokens, total_cost, latency, generation_time
  ↓
Frontend: onChunk callback accumulates chunks
  - Unescape newlines (\\n → \n)
  - Update Zustand chatStore via updateLastMessage action
  - Capture metadata via updateLastMessageMetadata action
  - UI updates automatically via Zustand subscriptions
  ↓
Frontend: Message component renders based on format
  - text: ReactMarkdown
  - json: renderJsonAsTree() with collapsible raw view
  - xml: renderXmlAsTree() with collapsible raw view
  - Shows model, temperature, tokens, cost, latency, generation time
```

### Authentication Flow

- Login/Register: POST to /api/login or /api/register with credentials
- Validation: Via `validation.AuthRequestValidator` (username, email, password rules)
- Response: JWT token (24-hour expiration, configurable via env)
- Storage: localStorage.getItem('auth_token')
- Protected routes: Authorization: Bearer {token} header
- User extracted from JWT claims via context value
- JWT secret: **Must be 32+ characters** (validated on startup)

## Key Technical Decisions

### Clean Layered Architecture (New)
The backend follows a **service-oriented pattern** with clear separation:

**API Layer** (`internal/api/handlers/`):
- `auth_handlers.go`: Login, register endpoints
- `chat_handlers.go`: Chat streaming, conversation management, summarization
- `models_handlers.go`: Model configuration endpoints
- HTTP-specific concerns only (request parsing, response formatting)

**Service Layer** (`internal/service/`):
- `chat_service.go`: Chat business logic, message streaming
- `conversation_service.go`: Conversation CRUD operations
- `summary_service.go`: Summarization strategies (first, progressive)
- `openrouter_provider.go`: OpenRouter LLM integration
- `genkit_provider.go`: Firebase Genkit integration (experimental)
- Pure business logic, no HTTP or database details

**Repository Layer** (`internal/repository/`):
- `repository.go`: Interface definitions for all database operations
- `postgres_repository.go`: PostgreSQL implementation of interfaces
- Abstraction allows easy testing and database swapping

**Validation Layer** (`pkg/validation/`):
- `auth_validator.go`: Username (3-50 chars), email, password (8+ chars) validation
- `chat_validator.go`: Message length, temperature (0.0-2.0), format, War and Peace percentage (1-100)
- Comprehensive test coverage in `*_test.go` files

**Configuration Layer** (`internal/config/`):
- `app.go`: Centralized AppConfig loading from environment
- `models.go`: Model configuration loader and validator
- Loads all config on startup into single struct

### Database Migrations with golang-migrate (New)
Unlike manual schema creation, the project **now uses migrations**:

**Migration Files** (`backend/migrations/`):
- `000001_create_users_table.up.sql` / `.down.sql`
- `000002_create_conversations_table.up.sql` / `.down.sql`
- `000003_create_messages_table.up.sql` / `.down.sql`
- `000004_create_conversation_summaries_table.up.sql` / `.down.sql`
- `000005_add_cost_tracking_to_messages.up.sql` / `.down.sql`
- `000006_add_provider_to_messages.up.sql` / `.down.sql`
- `000007_add_war_and_peace_fields.up.sql` / `.down.sql`

**Auto-Application**:
- Migrations run automatically on backend startup
- `postgres.RunMigrations()` in `cmd/server/main.go`
- Version tracking in `schema_migrations` table
- Bidirectional support (up/down) for rollbacks

**Adding New Migrations**:
```bash
# Create new migration files
migrate create -ext sql -dir backend/migrations -seq your_migration_name

# Edit .up.sql and .down.sql files
# Restart backend to auto-apply
docker compose restart backend
```

### Dual LLM Provider Support (New)
The application supports **two LLM providers** via strategy pattern:

**OpenRouter Provider** (production, default):
- Direct HTTP API integration
- Cost tracking via generation IDs
- Async cost fetching with exponential backoff (3 retries)
- Supports all models in `config/models.json`

**Genkit Provider** (experimental):
- Firebase Genkit framework integration
- OpenAI-compatible API layer
- Same interface as OpenRouter provider
- Selected via `provider` field in chat requests

**Provider Selection**:
```typescript
// Frontend sends provider preference
POST /api/chat/stream {
  message: "Hello",
  provider: "openrouter"  // or "genkit"
}

// Backend uses appropriate provider
if req.Provider == "genkit" {
  provider = chatService.genkitProvider
} else {
  provider = chatService.openRouterProvider
}
```

**Provider Tracking**:
- Each message records which provider generated it
- Stored in `messages.provider` column (added in migration 000006)
- Displayed in UI alongside model name

### Vite Build System (New)
Frontend built with **Vite** instead of Create React App for superior performance:

**Build Performance**:
- **87% faster builds**: 11-12s → 1.67s
- **84% fewer dependencies**: 1,488 → 227 packages
- **100% vulnerability reduction**: 27 → 0
- **6.7% smaller bundle**: 116.72 KB → 108.88 KB gzipped

**Configuration** (`vite.config.ts`):
- Dev server on port 3000 with proxy to backend
- Hot Module Replacement (HMR) for instant updates
- Code splitting: react-vendor, markdown, zustand chunks
- Source maps enabled for debugging

**Environment Variables**:
- Prefix: `VITE_` (not `REACT_APP_`)
- Access: `import.meta.env.VITE_API_URL`
- Type definitions in `src/vite-env.d.ts`

**Build Output**:
- Output directory: `build/` (same as CRA)
- Production optimizations: minification, tree-shaking
- Served via nginx in Docker container

**Benefits**:
- Instant dev server startup
- Fast HMR without full reload
- Native ES modules support
- Better TypeScript integration
- Smaller production bundles

### War and Peace Context Injection
Advanced feature for **testing context window handling**:

**Implementation** (`internal/context/warandpeace.go`):
- Loads War and Peace text file (3.3MB) into memory on startup
- Provides `GetWarAndPeaceContext(percentage int)` function
- Returns specified percentage (1-100%) of the text
- Used to test how models handle large context windows

**Usage Flow**:
```typescript
// User enables in Settings UI
POST /api/chat/stream {
  message: "Analyze this text",
  use_war_and_peace: true,
  war_and_peace_percent: 50  // 50% of War and Peace text
}

// Backend appends to system prompt
systemPrompt += "\n\n" + context.GetWarAndPeaceContext(50)
```

**Database Tracking**:
- `messages.war_and_peace_used` (boolean)
- `messages.war_and_peace_percent` (integer)
- Added in migration 000007

**Validation**:
- Percentage must be 1-100 (validated by `ChatRequestValidator`)
- Cannot use with response format JSON/XML (validation error)

### Cost and Performance Tracking (New)
Messages now track **comprehensive cost and performance metrics**:

**Database Fields** (added in migration 000005):
```sql
messages (
  prompt_tokens INTEGER,        -- Tokens in user prompt + history
  completion_tokens INTEGER,    -- Tokens in assistant response
  total_cost DECIMAL(10, 6),   -- Total cost in USD
  latency INTEGER,             -- Time to first token (ms)
  generation_time INTEGER      -- Total generation time (ms)
)
```

**Cost Fetching Flow**:
1. OpenRouter returns `generation_id` in streaming response (via `USAGE:` SSE event)
2. Backend extracts and stores generation_id
3. **Async goroutine** fetches cost data from OpenRouter API:
   - Retries 3 times with exponential backoff (100ms, 200ms, 400ms)
   - GET `https://openrouter.ai/api/v1/generation?id={generation_id}`
   - Parses token counts and cost from response
4. Updates message record in database

**Performance Metrics**:
- **Latency**: Time from request start to first token received
- **Generation Time**: Total time to complete streaming response
- Both measured in milliseconds

**Display**:
- All metadata sent to frontend via SSE USAGE event
- Frontend displays model, temperature, tokens, cost, latency, and generation time
- Real-time updates during streaming using Zustand state management
- Useful for analyzing model efficiency and costs

### SSE Streaming Over WebSocket
- Uses Server-Sent Events (SSE) instead of WebSocket for simpler, unidirectional streaming
- Newlines escaped on backend (`\n` → `\\n`) and unescaped on frontend for protocol compliance
- Metadata sent as special SSE events with prefixes:
  - `CONV_ID:uuid-string` - Conversation ID for new conversations
  - `MODEL:model-name` - Model used for generating response
  - `TEMPERATURE:0.70` - Temperature used for generating response
  - `USAGE:{json}` - Complete usage metadata with tokens, cost, latency, generation_time

### Response Format System
- **Three formats supported**: text (default), JSON, XML
- **Format locking**: Once a conversation starts, format cannot be changed (stored in DB)
- **Schema support**: JSON and XML formats require schema definition
- **Format-specific system prompts**:
  - Text: Uses custom user prompt from localStorage
  - JSON: Hardcoded schema-enforcement prompt + user schema
  - XML: Hardcoded schema-enforcement prompt + user schema
- **Visual rendering**:
  - Text: ReactMarkdown with remark-gfm
  - JSON: Hierarchical tree structure supporting unlimited nesting depth + collapsible raw JSON view
  - XML: Hierarchical tree structure with syntax highlighting + collapsible raw XML view

### LLM Parameter Management
- **Temperature Control**: User-adjustable via Settings UI (0.0-2.0 slider, default 0.7)
  - Temperature sent with every request to LLM API
  - Saved with each message in database
  - Displayed alongside model name in message UI
  - Preference persisted in localStorage
  - Validated by `ChatRequestValidator` (0.0-2.0 range)
- **Format-aware parameters**: Different top_p/top_k for text vs structured formats
- **Environment-based configuration** (top_p and top_k only):
  - `OPENROUTER_TEXT_TOP_P/TOP_K` for text conversations (0.9/40)
  - `OPENROUTER_STRUCTURED_TOP_P/TOP_K` for JSON/XML (0.8/20)
- **Provider routing**: `require_parameters: true` ensures OpenRouter routes to providers that support all parameters
- **Automatic selection**: Backend chooses top_p/top_k based on conversation.ResponseFormat from DB

### Model Selection System
- **Configuration-based**: Available models defined in `backend/config/models.json`
- **Model structure**: Each model has id, name, provider, and tier fields
- **Default selection**: First model in config file used as default
- **Backend validation**: Model IDs validated via `config.IsValidModel()` before API calls
- **Frontend fetching**: `GET /api/models` returns available models for UI dropdown
- **User selection**: Models chosen via Settings modal (⚙️) dropdown
- **Persistence**: Selected model saved to localStorage (`selectedModel`)
- **Per-message tracking**: Model name stored in `messages.model` column
- **Real-time display**: Model name shown next to "AI" label in chat messages
- **SSE metadata**: Model name sent via `MODEL:` prefix in streaming response
- **API override**: Optional `model` parameter in chat requests overrides default

**Current Models** (as of config):
1. `meta-llama/llama-3.3-8b-instruct:free` - Llama 3.3 8B (Meta, free) - **DEFAULT**
2. `google/gemini-2.5-flash` - Gemini 2.5 Flash (Google, paid)
3. `openai/gpt-5-mini` - GPT-5 Mini (OpenAI, paid)
4. `z-ai/glm-4.5-air:free` - GLM 4.5 Air (Z-AI, free)
5. `alibaba/tongyi-deepresearch-30b-a3b:free` - Tongyi DeepResearch 30B (Alibaba, free)
6. `openrouter/polaris-alpha` - Polaris Alpha (OpenRouter, paid)

Plus 4 additional models (check `config/models.json` for full list)

### Message History Pattern
- Full conversation history sent with every request to LLM for context coherence
- System prompt **always** prepended as first message: `{role: "system", content: "...prompt..."}`
- History retrieved in chronological order from repository
- **Summary optimization**: If conversation has active summary, only sends messages after last summarized message

### Frontend State Management (Zustand)
The frontend uses **Zustand** for centralized state management with three stores:

**Chat Store** (`src/stores/chatStore.ts`):
- Chat messages, conversation ID/title/format/schema, summaries
- Actions: addMessage, updateLastMessage, updateLastMessageMetadata, setConversationId, reset
- Real-time metadata updates during streaming (model, temperature, tokens, cost, latency)

**Settings Store** (`src/stores/settingsStore.ts`):
- Model selection, temperature, system prompt, response format/schema
- War and Peace settings, provider selection, settings modal state
- Persisted to localStorage via Zustand persist middleware

**Auth Store** (`src/stores/authStore.ts`):
- Authentication token, user info, login/logout actions
- Persisted to localStorage for session management

**Benefits**:
- Centralized state with minimal boilerplate
- Built-in devtools middleware for debugging
- Automatic localStorage persistence for settings and auth
- Type-safe with TypeScript interfaces
- No prop drilling - direct state access in any component

**Component separation**:
- Message.tsx handles format-specific rendering and displays metadata
- SettingsModal.tsx handles model selection and parameter configuration
- Chat.tsx orchestrates streaming and state updates

### UUID for All IDs
- All database IDs use PostgreSQL UUID type (Universally Unique Identifiers)
- Backend: Uses `github.com/google/uuid` v1.6.0 for UUID generation
- User IDs, Conversation IDs, and Message IDs are all UUIDs (string type in Go)
- Frontend: All ID types changed from `number` to `string` to accommodate UUID strings
- Benefits: Better distributed system support, higher collision resistance, cryptographic strength
- In SSE metadata, conversation IDs are sent as plain UUID strings (CONV_ID:uuid-string format)

## Database Schema

```sql
users (
  id UUID PRIMARY KEY,
  username VARCHAR UNIQUE,
  email VARCHAR,
  password_hash VARCHAR,
  created_at TIMESTAMP
)
  ↓
conversations (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users,
  title VARCHAR,
  response_format VARCHAR(10) DEFAULT 'text',    -- 'text', 'json', or 'xml'
  response_schema TEXT,                          -- Schema definition for structured formats
  active_summary_id UUID,                        -- References most recent summary (deprecated, query by created_at instead)
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)
  ↓
messages (
  id UUID PRIMARY KEY,
  conversation_id UUID REFERENCES conversations,
  role VARCHAR,
  content TEXT,
  model VARCHAR(255),                            -- LLM model used for assistant responses
  temperature REAL,                              -- Temperature used for generating response (0.0-2.0)
  provider VARCHAR(50),                          -- NEW: 'openrouter' or 'genkit'
  prompt_tokens INTEGER,                         -- NEW: Tokens in prompt + history
  completion_tokens INTEGER,                     -- NEW: Tokens in response
  total_cost DECIMAL(10, 6),                     -- NEW: Cost in USD
  latency INTEGER,                               -- NEW: Time to first token (ms)
  generation_time INTEGER,                       -- NEW: Total generation time (ms)
  war_and_peace_used BOOLEAN DEFAULT FALSE,      -- NEW: Was War and Peace context used
  war_and_peace_percent INTEGER,                 -- NEW: Percentage of text used (1-100)
  created_at TIMESTAMP
)
  ↓
conversation_summaries (
  id UUID PRIMARY KEY,
  conversation_id UUID REFERENCES conversations ON DELETE CASCADE,
  summary_content TEXT NOT NULL,                 -- LLM-generated summary of conversation
  summarized_up_to_message_id UUID REFERENCES messages ON DELETE SET NULL,
  usage_count INTEGER DEFAULT 0,                 -- Increments each use; triggers re-summarization at 2+
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
  ↓
schema_migrations (
  version BIGINT PRIMARY KEY,                    -- NEW: Migration version tracking
  dirty BOOLEAN NOT NULL                         -- NEW: Migration state
)
```

**Key Features:**
- All IDs are UUID type for distributed system compatibility
- Conversations auto-created on first message with first 100 chars as title
- **response_format** column locks format after first message (cannot be changed)
- **response_schema** stores JSON/XML schema definition for validation
- **model** and **provider** columns track which LLM generated each response
- **temperature** column tracks temperature setting (0.0-2.0) used for each response
- **Cost tracking fields**: prompt_tokens, completion_tokens, total_cost (added in migration 000005)
- **Performance tracking**: latency, generation_time in milliseconds
- **War and Peace tracking**: war_and_peace_used, war_and_peace_percent (added in migration 000007)
- **conversation_summaries** table stores LLM-generated conversation summaries
- **schema_migrations** table tracks applied migrations for golang-migrate
- Cascade deletes prevent orphaned records
- Indexes on user_id, conversation_id, and conversation_summaries.conversation_id

## Configuration

### Environment Variables (.env)

**Required:**
- `OPENROUTER_API_KEY` - API key for OpenRouter LLM service
- `JWT_SECRET` - **Must be 32+ characters** (validated on startup)

**Optional (with defaults):**
- `SERVER_PORT` - Backend server port (default: 8080)
- `OPENROUTER_SYSTEM_PROMPT` - Default system prompt (default: "You are a helpful assistant.")
- **Format-Aware LLM Parameters** (temperature is user-controlled via UI):
  - `OPENROUTER_TEXT_TOP_P` - Top-P for text conversations (default: 0.9)
  - `OPENROUTER_TEXT_TOP_K` - Top-K for text conversations (default: 40)
  - `OPENROUTER_STRUCTURED_TOP_P` - Top-P for JSON/XML (default: 0.8)
  - `OPENROUTER_STRUCTURED_TOP_K` - Top-K for JSON/XML (default: 20)
- `JWT_EXPIRATION_HOURS` - Token expiration in hours (default: 24)
- `DB_HOST` - PostgreSQL host (default: postgres in Docker, localhost locally)
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_USER` - PostgreSQL user (default: postgres)
- `DB_PASSWORD` - PostgreSQL password (default: postgres)
- `DB_NAME` - Database name (default: chatapp)
- `DB_SSLMODE` - SSL mode (default: disable)
- `VITE_API_URL` - Backend URL for frontend (default: http://localhost:8080, used by Vite)

**Deprecated:**
- `OPENROUTER_MODEL` - Model selection now via `backend/config/models.json`

### Model Configuration (backend/config/models.json)

Available LLM models are configured in a JSON file with the following structure:

```json
[
  {
    "id": "meta-llama/llama-3.3-8b-instruct:free",
    "name": "Llama 3.3 8B Instruct (Free)",
    "provider": "Meta",
    "tier": "free"
  }
]
```

**Configuration Details:**
- **id**: OpenRouter model identifier (used in API calls)
- **name**: Display name shown in UI
- **provider**: Company/organization that created the model
- **tier**: "free" or "paid" (for UI organization/filtering)

**Default Model**: The first model in the array is used as the default when no model is explicitly selected.

**Loading**: Configuration is loaded on backend startup via `config.LoadModels()` in `cmd/server/main.go`.

**Validation**: Backend validates model IDs via `config.IsValidModel(modelID)` before making API calls.

### Demo User
Automatically seeded on backend startup (idempotent):
- Username: `demo`
- Password: `demo123`

## Common Development Tasks

### Adding a New API Endpoint

**1. Add handler to appropriate file** (`internal/api/handlers/`):
```go
func (h *ChatHandlers) NewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Extract user from context
    user := auth.GetUserFromContext(r.Context())

    // Validate request
    var req NewRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Call service layer
    result, err := h.chatService.DoSomething(user.ID, req.Data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Return response
    json.NewEncoder(w).Encode(result)
}
```

**2. Register route in `cmd/server/main.go`**:
```go
mux.HandleFunc("POST /api/new/endpoint", enableCORS(auth.AuthMiddleware(chatHandlers.NewEndpoint)))
mux.HandleFunc("OPTIONS /api/new/endpoint", corsHandler)  // ← CORS preflight
```

**3. Add business logic to service layer** (`internal/service/`):
```go
func (s *ChatService) DoSomething(userID string, data string) (Result, error) {
    // Business logic here
    // Call repository for database operations
    return s.repo.SomeQuery(userID, data)
}
```

**4. Add database operations to repository** (`internal/repository/`):
```go
// Add to repository.go interface
type Repository interface {
    SomeQuery(userID string, data string) (Result, error)
}

// Implement in postgres_repository.go
func (r *PostgresRepository) SomeQuery(userID string, data string) (Result, error) {
    // SQL query here
}
```

**5. Update frontend service** (`frontend/src/services/chat.ts`):
```typescript
async newEndpoint(data: string): Promise<Result> {
  const response = await fetch(`${API_URL}/api/new/endpoint`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getAuthToken()}`
    },
    body: JSON.stringify({ data })
  });
  return response.json();
}
```

### Adding a Database Migration

**1. Create migration files**:
```bash
cd backend
migrate create -ext sql -dir migrations -seq add_new_field
```

**2. Edit `.up.sql`**:
```sql
ALTER TABLE messages ADD COLUMN new_field VARCHAR(255);
CREATE INDEX idx_messages_new_field ON messages(new_field);
```

**3. Edit `.down.sql`**:
```sql
DROP INDEX idx_messages_new_field;
ALTER TABLE messages DROP COLUMN new_field;
```

**4. Restart backend** (auto-applies migrations):
```bash
docker compose restart backend
```

**5. Update repository interface and implementation**:
```go
// repository.go
type Message struct {
    ID       string
    NewField string  // Add new field
}

// postgres_repository.go - update queries to include new field
```

### Modifying LLM Behavior

**OpenRouter Provider** (`internal/service/openrouter_provider.go`):
- Modify `ChatWithHistoryStream()` for streaming logic
- Adjust `buildMessagesWithHistory()` for message formatting
- Update `ChatRequest` struct for new API parameters

**Genkit Provider** (`internal/service/genkit_provider.go`):
- Similar structure to OpenRouter provider
- Modify for Genkit-specific behavior

**System Prompts**:
- Default: Edit `OPENROUTER_SYSTEM_PROMPT` in `.env`
- Format-specific: Edit prompt construction in `ChatService.SendMessageStream()`

### Adding a New LLM Provider

**1. Create provider file** (`internal/service/new_provider.go`):
```go
type NewProvider struct {
    apiKey string
    config *config.LLMConfig
}

func NewNewProvider(apiKey string, cfg *config.LLMConfig) *NewProvider {
    return &NewProvider{apiKey: apiKey, config: cfg}
}

func (p *NewProvider) ChatWithHistoryStream(
    messages []Message,
    systemPrompt string,
    format string,
    modelID string,
    temperature float64,
) (<-chan string, error) {
    // Implement streaming logic
}
```

**2. Add to ChatService** (`internal/service/chat_service.go`):
```go
type ChatService struct {
    openRouterProvider *OpenRouterProvider
    genkitProvider     *GenkitProvider
    newProvider        *NewProvider  // Add new provider
}
```

**3. Update handler** (`internal/api/handlers/chat_handlers.go`):
```go
var provider interface {
    ChatWithHistoryStream(...) (<-chan string, error)
}

switch req.Provider {
case "openrouter":
    provider = h.chatService.OpenRouterProvider
case "genkit":
    provider = h.chatService.GenkitProvider
case "newprovider":
    provider = h.chatService.NewProvider  // Add new case
default:
    provider = h.chatService.OpenRouterProvider
}
```

### Adding UI Components

**1. Create component** (`frontend/src/components/NewComponent.tsx`):
```typescript
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';
import { useChatStore, useSettingsStore } from '../stores';

export default function NewComponent() {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  // Access Zustand stores
  const { messages, addMessage } = useChatStore();
  const { selectedModel, setModel } = useSettingsStore();

  return (
    <div style={{ backgroundColor: colors.background }}>
      {/* Component content with access to global state */}
    </div>
  );
}
```

**2. Import in parent component**:
```typescript
import NewComponent from './NewComponent';

// Use in JSX
<NewComponent />
```

**3. Follow existing patterns**:
- Use `getTheme()` for consistent theming
- Use Zustand stores for state management (auto-persisted to localStorage)
- Access stores via hooks: `useChatStore()`, `useSettingsStore()`, `useAuthStore()`
- Update state via store actions, not direct mutations
- Use fetch with JWT token for API calls

## Security Notes

**Current Implementation:**
- JWT secret **required to be 32+ characters** (validated on startup)
- CORS allows all origins (`Access-Control-Allow-Origin: *`)
- Bcrypt password hashing with default cost factor
- User ownership verified for conversation access
- Demo credentials valid for testing only
- SQL injection protection via parameterized queries

**Production Improvements Needed:**
- Move JWT secret to secrets manager (Vault, AWS Secrets Manager)
- Restrict CORS to specific frontend origin
- Add rate limiting on auth endpoints
- Enable HTTPS/TLS
- Implement refresh tokens
- Add request logging and monitoring
- Enable database connection pooling limits

## Performance Considerations

- Message history sent with every request (use summaries for large conversations)
- No caching of LLM responses (every message goes to provider)
- Single database connection pool (tuning available via driver)
- Frontend renders all messages in DOM (virtualizing list recommended for 1000+ messages)
- SSE streaming prevents browser from accumulating large response objects
- Cost data fetched asynchronously (doesn't block response)
- War and Peace context loads once on startup (cached in memory)

## Deployment

```bash
# Set up environment
cp .env.example .env
# Edit .env with your OPENROUTER_API_KEY and JWT_SECRET (32+ chars)

# Build and run
docker compose build
docker compose up

# Access
# Frontend: http://localhost:3000
# Backend health check: http://localhost:8080/api/health

# Check logs
docker compose logs backend
docker compose logs frontend
docker compose logs postgres

# Clean restart (removes volumes)
docker compose down -v
docker compose up --build
```

**Docker Images:**
- Frontend: nginx:alpine serving React build with SPA routing fallback
- Backend: alpine:latest with Go binary, ca-certificates for HTTPS
- PostgreSQL: postgres:15-alpine with persistent data volume

## Testing Infrastructure

**Frontend:**
- Vitest + React Testing Library (Vite-native testing)
- Command: `cd frontend && npm test`
- Component tests in same directory as components
- Fast test execution with HMR
- No snapshot tests configured

**Backend:**
- Standard Go testing (`go test ./...`)
- Validation layer has comprehensive test coverage
- Command: `cd backend && go test ./pkg/validation -v`
- Database tests need PostgreSQL running or mocking
- No mocking/stubbing framework currently in use

**Test Examples:**
```bash
# Run all backend tests
cd backend && go test ./...

# Run specific package with verbose output
cd backend && go test ./pkg/validation -v

# Run frontend tests
cd frontend && npm test

# Run frontend tests in CI mode
cd frontend && CI=true npm test
```

## Routing (Go 1.25.3 Native)

The backend uses Go 1.22+ native HTTP routing with method-based patterns:

```go
// Method-based routing syntax (Go 1.22+)
mux.HandleFunc("GET /api/health", handler)
mux.HandleFunc("POST /api/login", handler)
mux.HandleFunc("GET /api/conversations/{id}/messages", handler)
mux.HandleFunc("DELETE /api/conversations/{id}", handler)
```

Path parameters are extracted using `r.PathValue("id")`. This provides type-safe routing without third-party routers.

## CORS Support

Go 1.22+ method-based routing is method-specific, meaning routes like `"POST /api/login"` only match POST requests. However, browsers send preflight **OPTIONS requests** before actual cross-origin requests.

The backend explicitly registers OPTIONS handlers for all endpoints:

```go
mux.HandleFunc("POST /api/login", enableCORS(handler))
mux.HandleFunc("OPTIONS /api/login", corsHandler)  // ← Preflight response
```

This ensures:
- Browser sends OPTIONS (preflight) → server responds with CORS headers
- Browser then sends actual POST/GET/DELETE → request is allowed
- Frontend can make API calls from http://localhost:3000 to http://localhost:8080

**When adding new endpoints:**
1. Register the method route: `mux.HandleFunc("POST /api/new", handler)`
2. Register the OPTIONS route: `mux.HandleFunc("OPTIONS /api/new", corsHandler)`

## File Organization Reference

```
backend/
  cmd/server/main.go              # Entry point, route setup, server start, config loading
  config/models.json              # Available LLM models configuration
  migrations/                     # Database migration files (golang-migrate)
    000001_create_users_table.up.sql / .down.sql
    000002_create_conversations_table.up.sql / .down.sql
    ...
  internal/
    api/handlers/                 # HTTP handlers (auth, chat, models)
      auth_handlers.go
      chat_handlers.go
      models_handlers.go
    service/                      # Business logic layer
      chat_service.go
      conversation_service.go
      summary_service.go
      openrouter_provider.go
      genkit_provider.go
    repository/                   # Database abstraction layer
      repository.go               # Interface definitions
      postgres_repository.go      # PostgreSQL implementation
    config/                       # Configuration loading
      app.go                      # AppConfig struct and loader
      models.go                   # Models config loader
    context/                      # Context utilities
      warandpeace.go              # War and Peace text loader
    app/                          # App-level setup
      app.go                      # Application initialization
    auth/                         # Authentication
      auth.go                     # JWT generation and validation
      middleware.go               # Auth middleware
    logger/                       # Structured logging
      logger.go                   # Logrus setup
  pkg/validation/                 # Request validators
    auth_validator.go
    auth_validator_test.go
    chat_validator.go
    chat_validator_test.go

frontend/
  index.html                      # Vite entry point (root level, not in public/)
  vite.config.ts                  # Vite configuration (dev server, build, code splitting)
  src/
    components/                   # React components
      Chat.tsx                    # Main chat UI
      Message.tsx                 # Message rendering
      Login.tsx                   # Auth forms
      SettingsModal.tsx           # Settings UI
      Sidebar.tsx                 # Conversation sidebar
      Chat/                       # Chat subcomponents
        ChatHeader.tsx
        ChatMessages.tsx
        ChatInput.tsx
    services/                     # API clients
      auth.ts                     # JWT token management
      chat.ts                     # Chat API calls, SSE parsing
    stores/                       # Zustand state management
      chatStore.ts                # Chat messages and conversation state
      settingsStore.ts            # User preferences and settings
      authStore.ts                # Authentication state
      index.ts                    # Store exports
    contexts/                     # React contexts
      ThemeContext.tsx            # Theme state provider
    themes.ts                     # Color palettes
    vite-env.d.ts                 # Vite TypeScript definitions
    App.tsx                       # Root component
    index.tsx                     # Entry point

docker-compose.yml                # Service orchestration
.env.example                      # Configuration template
CLAUDE.md                         # This file
```

## Debugging Tips

**Backend:**
- Check logs: `docker compose logs backend --follow`
- Database connection: Verify DB_HOST, DB_PORT, DB_USER, DB_PASSWORD in .env
- LLM errors: Check OPENROUTER_API_KEY validity and quota
- Auth failures: Verify JWT_SECRET is 32+ characters
- Migration issues: Check `schema_migrations` table for version
- Cost tracking: Check for generation_id in SSE stream

**Frontend:**
- Browser console: Check for fetch errors, SSE connection issues
- Network tab: Inspect request/response headers and SSE stream format
- localStorage: Verify auth_token, theme, systemPrompt, selectedModel keys persist
- Theme not loading: Ensure ThemeContext loads before Chat component mounts

**Database:**
- Connect directly: `docker compose exec postgres psql -U postgres -d chatapp`
- Check tables: `\dt` in psql
- View migrations: `SELECT * FROM schema_migrations;`
- Clear data: `DELETE FROM conversations;` (cascade deletes messages)
- Reset demo user: `DELETE FROM users WHERE username = 'demo';` then restart backend

**Migrations:**
- Check status: `SELECT * FROM schema_migrations;`
- Force version (if stuck): `UPDATE schema_migrations SET version = N, dirty = false;`
- Manual migration: Use `migrate` CLI with connection string

## LLM Integration Details

### OpenRouter API (Primary)
- Endpoint: https://openrouter.ai/api/v1/chat/completions
- Authentication: Bearer token in Authorization header
- Default model: First model from `backend/config/models.json`
- Stream support: Full SSE streaming capability
- System prompt: First message with role='system'
- Cost tracking: Via generation IDs with async fetching

### Genkit API (Experimental)
- Firebase Genkit framework integration
- OpenAI-compatible API layer
- Same interface as OpenRouter provider
- Selected via `provider: "genkit"` in requests

### Model Selection
- **Configuration**: Models defined in `backend/config/models.json`
- **API Endpoint**: `GET /api/models` returns available models
- **Runtime Selection**: User chooses model from Settings UI dropdown
- **Override**: Optional `model` parameter in chat request overrides default
- **Validation**: Backend validates model ID via `config.IsValidModel()` before API calls
- **Tracking**: Model name and provider saved in `messages` table
- **Display**: Model name shown in UI next to "AI" label

### Response Format (Streaming)
```
data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"delta":{"content":" world"}}]}
data: MODEL:meta-llama/llama-3.3-8b-instruct:free
data: TEMPERATURE:0.70
data: USAGE:{"prompt_tokens":150,"completion_tokens":25,"total_tokens":175,"total_cost":0.000042,"latency":234,"generation_time":1250}
data: [DONE]
```

### Metadata Events (SSE with prefixes)
- `CONV_ID:uuid-string` - Conversation ID for new conversations
- `MODEL:model-name` - Model used for generating response
- `TEMPERATURE:0.70` - Temperature used for generating response (0.0-2.0)
- `USAGE:{json}` - Complete usage metadata including:
  - `prompt_tokens`: Tokens in prompt + history (required)
  - `completion_tokens`: Tokens in assistant response (required)
  - `total_tokens`: Sum of prompt and completion tokens (required)
  - `total_cost`: Cost in USD (optional, if available from provider)
  - `latency`: Time to first token in milliseconds (optional)
  - `generation_time`: Total generation time in milliseconds (optional)

Parsed by frontend: unescape newlines, collect chunks, capture metadata via Zustand actions, skip [DONE]

## Conversation Summarization Feature

### Overview
The application supports manual conversation summarization with progressive re-summarization. This allows long conversations to use summaries instead of full message history as LLM context, reducing token usage while maintaining conversation coherence.

### How It Works

**User Trigger:**
- User clicks 📝 button in chat interface
- Sends POST request to `/api/conversations/{id}/summarize`

**First Summarization:**
1. Backend checks if active summary exists via repository
2. If no summary: Retrieves all messages from conversation
3. Calls LLM provider with summarization-only system prompt
4. Creates new summary in database
5. Returns summary content and `summarized_up_to_message_id`

**Progressive Re-Summarization (after 2+ uses):**
1. Backend detects `activeSummary.UsageCount >= 2`
2. Builds input: `[old summary as context] + [messages after last summarized message]`
3. Calls LLM to create new summary from combined content
4. Saves new summary to database (old summary remains for history)
5. Returns updated summary

**Usage in Chat:**
- When user sends new message, handler checks for active summary
- If summary exists:
  - Fetches only messages AFTER `summarized_up_to_message_id`
  - Prepends summary as context to system prompt
  - Increments summary `usage_count`
- LLM receives: `[system prompt with summary context] + [recent messages only]`

### System Prompt Scenarios

The implementation carefully separates three system prompt scenarios:

1. **Normal Chat** (no summary):
   ```
   [default system prompt] + [user custom prompt] + [full conversation history]
   ```

2. **Summarization Request**:
   ```
   [summarization prompt ONLY] + [messages to summarize]
   ```
   No default system prompt or user custom prompt included

3. **Chat After Summary**:
   ```
   [summary as context] + [default system prompt] + [user custom prompt] + [recent messages only]
   ```

### Key Technical Details

- **Multiple summaries supported**: Each re-summarization creates a new row in `conversation_summaries`
- **Most recent summary**: Retrieved via `ORDER BY created_at DESC LIMIT 1`
- **Usage tracking**: `usage_count` incremented on each chat message sent after summary exists
- **Collapsible UI**: Uses HTML5 `<details>` element for expandable summary view
- **Persistence**: All summaries stored in database, loaded on conversation open
