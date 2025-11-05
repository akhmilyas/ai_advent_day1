# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Commands

### Docker Compose (Recommended)
```bash
docker compose build      # Build both frontend and backend images
docker compose up         # Start all services (PostgreSQL, Backend, Frontend)
docker compose down       # Stop all services
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
```bash
cd frontend
npm install
npm run build             # Production build
npm start                 # Development server (http://localhost:3000)
npm test                  # Run tests
```

### Testing
```bash
# Frontend tests
cd frontend && npm test

# Backend tests
cd backend && go test ./...
cd backend && go test ./internal/handlers -v  # Single package with verbose output
```

## Architecture Overview

### High-Level Design
This is a fullstack chat application with three main layers:

1. **Frontend (React/TypeScript)** on port 3000
   - Single-page app with login/register and chat interface
   - Connects to backend via HTTP REST (auth) and Server-Sent Events (chat streaming)
   - Client-side state: authentication tokens, theme preference, custom system prompts
   - All stored in localStorage

2. **Backend (Go)** on port 8080
   - REST API for authentication (/api/login, /api/register)
   - SSE streaming endpoint for chat responses (/api/chat/stream)
   - Middleware for JWT validation on protected routes
   - Manages conversation history and message persistence

3. **Database (PostgreSQL)** on port 5432
   - Three tables: users, conversations, messages
   - Cascading deletes for referential integrity
   - Indexes on foreign keys and frequently queried fields

### Data Flow for Chat (Streaming)

```
User sends message
  ↓
Frontend: ChatService.streamMessage()
  ↓
POST /api/chat/stream {message, conversation_id?, system_prompt?, response_format?, response_schema?}
  ↓
Backend: ChatStreamHandler
  - Validates JWT token
  - Gets/creates conversation (with format/schema if new)
  - Adds user message to DB
  - Fetches full conversation history
  - Builds format-specific system prompt (JSON/XML get schema instructions)
  ↓
Backend: llm.ChatWithHistoryStream(messages, systemPrompt, format)
  - Selects format-aware LLM parameters (temperature, top_p, top_k)
  - Builds message array: [system_prompt, user1, assistant1, ..., user_n]
  - Calls OpenRouter API with format-specific parameters
  - Streams response via SSE format (data: {chunk}\n\n)
  ↓
Frontend: onChunk callback accumulates chunks
  - Unescape newlines (\\n → \n)
  - Update UI incrementally
  ↓
Backend: Saves full response to DB after streaming completes
  ↓
Frontend: Message component renders based on format
  - text: ReactMarkdown
  - json: renderJsonAsTable() with collapsible raw view
  - xml: renderXmlAsTree() with collapsible raw view
```

### Authentication Flow

- Login/Register: POST to /api/login or /api/register with credentials
- Response: JWT token (24-hour expiration)
- Storage: localStorage.getItem('auth_token')
- Protected routes: Authorization: Bearer {token} header
- User extracted from JWT claims via context.Value(auth.UserContextKey)

## Key Technical Decisions

### SSE Streaming Over WebSocket
- Uses Server-Sent Events (SSE) instead of WebSocket for simpler, unidirectional streaming
- Newlines escaped on backend (`\n` → `\\n`) and unescaped on frontend for protocol compliance
- Metadata (conversation ID, model name) sent as special SSE events with prefixes (CONV_ID:, MODEL:)

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
  - JSON: Table with Key/Value columns + collapsible raw JSON view
  - XML: Tree structure with indentation + collapsible raw XML view

### LLM Parameter Management
- **Format-aware parameters**: Different params for text vs structured formats
- **Environment-based configuration**:
  - `OPENROUTER_TEXT_TEMPERATURE/TOP_P/TOP_K` for text conversations (0.7/0.9/40)
  - `OPENROUTER_STRUCTURED_TEMPERATURE/TOP_P/TOP_K` for JSON/XML (0.3/0.8/20)
- **No hardcoded constants**: All values come from environment variables
- **Automatic selection**: Backend chooses params based on conversation.ResponseFormat from DB

### Message History Pattern
- Full conversation history sent with every request to LLM for context coherence
- System prompt **always** prepended as first message: `{role: "system", content: "...prompt..."}`
- History retrieved in chronological order from DB

### Frontend State Management
- Minimal React state: messages, input, loading, conversationId, model, systemPrompt, responseFormat, responseSchema, conversationFormat, conversationSchema, theme, settingsOpen
- No Redux/complex state library; useContext for theme, useState for component local state
- localStorage for persistence: auth_token, theme, systemPrompt, responseFormat, responseSchema
- Component separation: Message.tsx handles format-specific rendering

### UUID for All IDs
- All database IDs use PostgreSQL UUID type (Universally Unique Identifiers)
- Backend: Uses `github.com/google/uuid` v1.3.0 for UUID generation
- User IDs, Conversation IDs, and Message IDs are all UUIDs (string type in Go)
- Frontend: All ID types changed from `number` to `string` to accommodate UUID strings
- Benefits: Better distributed system support, higher collision resistance, cryptographic strength
- In SSE metadata, conversation IDs are sent as plain UUID strings (CONV_ID:uuid-string format)

## Database Schema

```sql
users (id UUID PRIMARY KEY, username VARCHAR UNIQUE, email VARCHAR, password_hash VARCHAR, created_at TIMESTAMP)
  ↓
conversations (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users,
  title VARCHAR,
  response_format VARCHAR(10) DEFAULT 'text',    -- NEW: 'text', 'json', or 'xml'
  response_schema TEXT,                          -- NEW: Schema definition for structured formats
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)
  ↓
messages (id UUID PRIMARY KEY, conversation_id UUID REFERENCES conversations, role VARCHAR, content TEXT, created_at TIMESTAMP)
```

**Key Features:**
- All IDs are UUID type for distributed system compatibility and collision resistance
- Conversations auto-created on first message with first 100 chars as title
- **response_format** column locks format after first message (cannot be changed)
- **response_schema** stores JSON/XML schema definition for validation
- Messages store role ('user' or 'assistant') for history reconstruction
- Cascade deletes prevent orphaned records
- Indexes on user_id and conversation_id for query performance
- UUIDs generated on the backend using `google/uuid` package
- COALESCE used in queries to handle NULL values: `COALESCE(response_format, 'text')`

## Configuration

### Environment Variables (.env)

**Required:**
- `OPENROUTER_API_KEY` - API key for OpenRouter LLM service

**Optional (with defaults):**
- `OPENROUTER_MODEL` - Model ID (default: meta-llama/llama-3.3-8b-instruct:free)
- `OPENROUTER_SYSTEM_PROMPT` - Default system prompt (default: "You are a helpful assistant.")
- **Format-Aware LLM Parameters**:
  - `OPENROUTER_TEXT_TEMPERATURE` - Temperature for text conversations (default: 0.7)
  - `OPENROUTER_TEXT_TOP_P` - Top-P for text conversations (default: 0.9)
  - `OPENROUTER_TEXT_TOP_K` - Top-K for text conversations (default: 40)
  - `OPENROUTER_STRUCTURED_TEMPERATURE` - Temperature for JSON/XML (default: 0.3)
  - `OPENROUTER_STRUCTURED_TOP_P` - Top-P for JSON/XML (default: 0.8)
  - `OPENROUTER_STRUCTURED_TOP_K` - Top-K for JSON/XML (default: 20)
- `DB_HOST` - PostgreSQL host (default: postgres in Docker, localhost locally)
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_USER` - PostgreSQL user (default: postgres)
- `DB_PASSWORD` - PostgreSQL password (default: postgres)
- `DB_NAME` - Database name (default: chatapp)
- `DB_SSLMODE` - SSL mode (default: disable)
- `REACT_APP_API_URL` - Backend URL for frontend (default: http://localhost:8080)

### Demo User
Automatically seeded on backend startup:
- Username: `demo`
- Password: `demo123`

## Common Development Tasks

### Adding a New Chat Endpoint
1. Add handler function to `backend/internal/handlers/chat.go`
2. Register route in `backend/cmd/server/main.go` using Go 1.22+ method-based routing:
   ```go
   mux.HandleFunc("POST /api/new/endpoint", enableCORS(auth.AuthMiddleware(chatHandler.Handler)))
   mux.HandleFunc("OPTIONS /api/new/endpoint", corsHandler)  // ← CORS preflight
   ```
3. Extract path parameters with `r.PathValue("param_name")` if needed
4. Add corresponding method to ChatService in `frontend/src/services/chat.ts`
5. Update Chat component callbacks or UI if needed

### Modifying LLM Behavior
- Edit `backend/internal/llm/openrouter.go`:
  - Change system prompt in `GetSystemPrompt()`
  - Modify message building in `buildMessagesWithHistory()`
  - Adjust OpenRouter request parameters in ChatRequest struct

### Adding UI Components
- Create component in `frontend/src/components/`
- Import in Chat.tsx and add to return JSX
- Theme colors accessible via `getTheme(theme === 'dark')` returns color object
- Leverage existing styles object for consistent spacing/typography

### Database Migrations
- Currently no migration tool (Migrate, Flyway, etc.)
- Schema created in `backend/internal/db/postgres.go` InitDB() function
- Add new tables to InitDB() and restart backend
- **Production note:** Use proper migration tool before scaling

## Security Notes

**Current Implementation:**
- JWT secret hardcoded in `backend/internal/auth/auth.go` (not env-configurable)
- CORS allows all origins (`Access-Control-Allow-Origin: *`)
- Bcrypt password hashing with default cost factor
- User ownership verified for conversation access
- Demo credentials valid for testing only

**Production Improvements Needed:**
- Move JWT secret to environment variable
- Restrict CORS to specific frontend origin
- Add rate limiting on auth endpoints
- Enable HTTPS
- Use secrets management (Vault, AWS Secrets Manager)

## Performance Considerations

- Message history sent with every request (could paginate for large conversations)
- No caching of LLM responses (every message goes to OpenRouter)
- Single database connection pool (tuning available via driver)
- Frontend renders all messages in DOM (virtualizing list for 1000+ messages recommended)
- SSE streaming prevents browser from accumulating large response objects

## Deployment

```bash
# Set up environment
cp .env.example .env
# Edit .env with your OPENROUTER_API_KEY

# Build and run
docker compose build
docker compose up

# Access
# Frontend: http://localhost:3000
# Backend health check: http://localhost:8080/api/health
```

**Docker Images:**
- Frontend: nginx:alpine serving React build with SPA routing fallback
- Backend: alpine:latest with Go binary, ca-certificates for HTTPS
- PostgreSQL: postgres:15-alpine with data volume

## Testing Infrastructure

**Frontend:**
- Jest + React Testing Library (via react-scripts)
- No snapshot tests configured
- Component tests in same directory as components

**Backend:**
- Standard Go testing (go test)
- Database tests need PostgreSQL running or mocking
- No mocking/stubbing framework currently in use

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
  cmd/server/main.go              # Entry point, route setup (Go 1.22+ method-based routing), server start
  internal/auth/auth.go           # JWT, login, register, middleware
  internal/db/                    # Database layer (postgres.go, user.go, conversation.go)
  internal/handlers/chat.go       # HTTP handlers for /api/chat endpoints
  internal/llm/openrouter.go      # LLM integration, message building, streaming

frontend/
  src/App.tsx                     # Root, auth routing
  src/components/Chat.tsx         # Main chat UI, message streaming
  src/components/Login.tsx        # Auth forms
  src/components/SettingsModal.tsx # System prompt configuration
  src/services/auth.ts            # JWT token management
  src/services/chat.ts            # API calls, SSE parsing
  src/contexts/ThemeContext.tsx   # Theme state provider
  src/themes.ts                   # Color palettes

docker-compose.yml                # Service orchestration
.env.example                      # Configuration template
```

## Debugging Tips

**Backend:**
- Check logs: `docker compose logs backend`
- Database connection: Verify DB_HOST, DB_PORT, DB_USER, DB_PASSWORD in .env
- LLM errors: Check OPENROUTER_API_KEY validity and quota
- Auth failures: Verify JWT secret matches between sign/verify

**Frontend:**
- Browser console: Check for fetch errors, SSE connection issues
- Network tab: Inspect request/response headers and SSE stream format
- localStorage: Verify auth_token, theme, systemPrompt keys persist
- Theme not loading: Ensure ThemeContext loads before Chat component mounts

**Database:**
- Connect directly: `psql -h localhost -U postgres -d chatapp`
- Check tables: `\dt` in psql
- Clear data: `DELETE FROM conversations;` (cascade deletes messages)
- Reset demo user: Stop backend, delete user, restart backend

## LLM Integration Details

**OpenRouter API:**
- Endpoint: https://openrouter.ai/api/v1/chat/completions
- Authentication: Bearer token in Authorization header
- Default model: meta-llama/llama-3.3-8b-instruct:free (free tier)
- Stream support: Full SSE streaming capability
- System prompt: First message with role='system'

**Supported Models:**
- Free models available at openrouter.ai
- Change via OPENROUTER_MODEL environment variable
- Model name returned in every response for UI display

**Response Format (Streaming):**
```
data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"delta":{"content":" world"}}]}
data: [DONE]
```

Parsed by frontend: unescape newlines, collect chunks, skip [DONE]

## Response Format Feature

### Overview
The application supports three response formats:
1. **Text** (default) - Natural conversation with markdown
2. **JSON** - Structured data with schema enforcement
3. **XML** - Structured markup with schema enforcement

### Format Selection Flow

**New Conversation:**
1. User opens Settings (⚙️) before sending first message
2. Selects format (Text/JSON/XML) via radio buttons
3. If JSON/XML selected, must provide schema in textarea
4. Sends first message → format + schema saved to DB
5. Format is now **locked** for this conversation

**Existing Conversation:**
1. User opens Settings (⚙️)
2. Sees "🔒 Locked Configuration" message with current format
3. Radio buttons are hidden (not disabled)
4. Schema is shown in read-only textarea
5. Cannot change format or schema

### Frontend Implementation

**Components:**
- `SettingsModal.tsx`: Format/schema configuration UI
  - Shows radio buttons only for new conversations (`!isExistingConversation`)
  - Locks configuration for existing conversations
  - Stores format/schema in localStorage (for new conversations)
  - Reads from `conversationFormat`/`conversationSchema` props (for existing)

- `Message.tsx`: Format-specific rendering
  - `renderJsonAsTable()`: Parses JSON, displays as Key/Value table with collapsible raw view
  - `renderXmlAsTree()`: Parses XML with DOMParser, displays as indented tree with collapsible raw view
  - Uses `<details>/<summary>` HTML elements for collapsible raw views

**State Management:**
```typescript
// User preferences (new conversations)
const [responseFormat, setResponseFormat] = useState<ResponseFormat>('text');
const [responseSchema, setResponseSchema] = useState<string>('');

// Locked conversation settings (existing conversations)
const [conversationFormat, setConversationFormat] = useState<ResponseFormat | null>(null);
const [conversationSchema, setConversationSchema] = useState<string>('');
```

### Backend Implementation

**Database:**
```sql
conversations (
  response_format VARCHAR(10) DEFAULT 'text',
  response_schema TEXT
)
```

**Handler Logic:**
```go
// Create new conversation
if req.ConversationID == "" {
  conversation, err = db.CreateConversation(user.ID, title, req.ResponseFormat, req.ResponseSchema)
}

// Build format-specific system prompt
if conversation.ResponseFormat == "json" && conversation.ResponseSchema != "" {
  effectiveSystemPrompt = fmt.Sprintf("You must respond ONLY with valid JSON that matches this exact schema...")
} else if conversation.ResponseFormat == "xml" && conversation.ResponseSchema != "" {
  effectiveSystemPrompt = fmt.Sprintf("You must respond ONLY with valid XML that matches this exact schema...")
} else {
  effectiveSystemPrompt = req.SystemPrompt  // User's custom prompt
}

// Pass format to LLM for parameter selection
chunks, err := llm.ChatWithHistoryStream(currentHistory, effectiveSystemPrompt, conversation.ResponseFormat)
```

**LLM Parameter Selection:**
```go
func GetTemperature(format string) *float64 {
  if format == "json" || format == "xml" {
    return os.Getenv("OPENROUTER_STRUCTURED_TEMPERATURE")  // 0.3
  }
  return os.Getenv("OPENROUTER_TEXT_TEMPERATURE")  // 0.7
}
```

### Visual Rendering

**JSON Format:**
```
[View Raw JSON ▼]  ← Collapsible details element

┌─────────────────┬──────────────────────┐
│ Key             │ Value                │
├─────────────────┼──────────────────────┤
│ name            │ John Doe             │
│ age             │ 30                   │
│ address         │ {"city": "NYC", ...} │
└─────────────────┴──────────────────────┘
```

**XML Format:**
```
[View Raw XML ▼]  ← Collapsible details element

<response version="1.0">
  <question>
    What is AI?
  </question>
  <answer>
    Artificial Intelligence
  </answer>
</response>

↑ Rendered as indented tree with:
- Color-coded tags (primary color)
- Left border (3px solid)
- Alternating backgrounds
- Inline text for simple elements
```

### Key Features

1. **Format Locking**: Prevents format changes mid-conversation (data integrity)
2. **Schema Enforcement**: LLM instructed to strictly follow schema
3. **Visual Parsing**: JSON/XML parsed and rendered visually
4. **Raw View**: Collapsible access to original response
5. **Error Handling**: Falls back to `<pre>` if parsing fails
6. **Parameter Optimization**: Lower temperature for structured formats (0.3 vs 0.7)
