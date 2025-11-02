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
POST /api/chat/stream {message, conversation_id?, system_prompt?}
  ↓
Backend: ChatStreamHandler
  - Validates JWT token
  - Gets/creates conversation
  - Adds user message to DB
  - Fetches full conversation history
  ↓
Backend: llm.ChatWithHistoryStream()
  - Builds message array: [system_prompt, user1, assistant1, ..., user_n]
  - Calls OpenRouter API (https://openrouter.ai/api/v1/chat/completions)
  - Streams response via SSE format (data: {chunk}\n\n)
  ↓
Frontend: onChunk callback accumulates chunks
  - Unescape newlines (\\n → \n)
  - Update UI incrementally
  ↓
Backend: Saves full response to DB after streaming completes
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

### System Prompt Design
- User-provided prompts are **appended** to environment default: `"{env_default}\n\nAdditional instructions: {user_prompt}"`
- Session-wide (not per-conversation) setting stored in localStorage
- Passed per-request to allow dynamic changes without UI restart

### Message History Pattern
- Full conversation history sent with every request to LLM for context coherence
- System prompt **always** prepended as first message: `{role: "system", content: "...prompt..."}`
- History retrieved in chronological order from DB

### Frontend State Management
- Minimal React state: messages, input, loading, conversationId, model, systemPrompt, theme, settingsOpen
- No Redux/complex state library; useContext for theme, useState for component local state
- localStorage for persistence: auth_token, theme, systemPrompt

## Database Schema

```sql
users (id, username, email, password_hash, created_at)
  ↓
conversations (id, user_id, title, created_at, updated_at)
  ↓
messages (id, conversation_id, role, content, created_at)
```

**Key Features:**
- Conversations auto-created on first message with first 100 chars as title
- Messages store role ('user' or 'assistant') for history reconstruction
- Cascade deletes prevent orphaned records
- Indexes on user_id and conversation_id for query performance

## Configuration

### Environment Variables (.env)

**Required:**
- `OPENROUTER_API_KEY` - API key for OpenRouter LLM service

**Optional (with defaults):**
- `OPENROUTER_MODEL` - Model ID (default: meta-llama/llama-3.3-8b-instruct:free)
- `OPENROUTER_SYSTEM_PROMPT` - Default system prompt (default: "You are a helpful assistant.")
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
   mux.HandleFunc("GET /api/new/endpoint", enableCORS(auth.AuthMiddleware(chatHandler.Handler)))
   ```
3. Extract path parameters with `r.PathValue("param_name")` if needed
4. Update ChatService in `frontend/src/services/chat.ts` to call the endpoint
5. Update Chat component callbacks if needed

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
