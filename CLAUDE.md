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
   - Connects to backend via HTTP REST (auth, models) and Server-Sent Events (chat streaming)
   - Client-side state: authentication tokens, theme preference, custom system prompts, selected model, temperature
   - All stored in localStorage

2. **Backend (Go)** on port 8080
   - REST API for authentication (/api/login, /api/register)
   - REST API for model configuration (/api/models)
   - SSE streaming endpoint for chat responses (/api/chat/stream)
   - Middleware for JWT validation on protected routes
   - Manages conversation history, message persistence, and model tracking

3. **Database (PostgreSQL)** on port 5432
   - Four tables: users, conversations, messages, conversation_summaries
   - Cascading deletes for referential integrity
   - Indexes on foreign keys and frequently queried fields
   - Summary tracking with usage count for progressive re-summarization

### Data Flow for Chat (Streaming)

```
User sends message
  ↓
Frontend: ChatService.streamMessage()
  ↓
POST /api/chat/stream {message, conversation_id?, system_prompt?, response_format?, response_schema?, model?, temperature?}
  ↓
Backend: ChatStreamHandler
  - Validates JWT token
  - Validates model ID (if provided) against config
  - Gets/creates conversation (with format/schema if new)
  - Adds user message to DB
  - Fetches full conversation history
  - Builds format-specific system prompt (JSON/XML get schema instructions)
  ↓
Backend: llm.ChatWithHistoryStream(messages, systemPrompt, format, modelOverride, temperature)
  - Uses provided model or defaults to first model in config
  - Uses user-provided temperature (0.0-2.0)
  - Selects format-aware top_p and top_k from environment variables
  - Sets provider.require_parameters: true to ensure parameter support
  - Builds message array: [system_prompt, user1, assistant1, ..., user_n]
  - Calls OpenRouter API with selected model, temperature, top_p, top_k, and provider config
  - Streams response via SSE format (data: {chunk}\n\n)
  - Sends model name via SSE (MODEL: model-name)
  - Sends temperature via SSE (TEMPERATURE: 0.70)
  ↓
Frontend: onChunk callback accumulates chunks
  - Unescape newlines (\\n → \n)
  - Update UI incrementally
  - Capture model name from MODEL: event
  - Capture temperature from TEMPERATURE: event
  ↓
Backend: Saves full response to DB after streaming completes (with model and temperature fields)
  ↓
Frontend: Message component renders based on format
  - text: ReactMarkdown
  - json: renderJsonAsTree() with collapsible raw view
  - xml: renderXmlAsTree() with collapsible raw view
  - Shows model name and temperature next to "AI" label (e.g., "AI (model-name, temp: 0.70)")
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
  - JSON: Hierarchical tree structure supporting unlimited nesting depth + collapsible raw JSON view
  - XML: Hierarchical tree structure with syntax highlighting + collapsible raw XML view

### LLM Parameter Management
- **Temperature Control**: User-adjustable via Settings UI (0.0-2.0 slider, default 0.7)
  - Temperature sent with every request to OpenRouter API
  - Saved with each message in database
  - Displayed alongside model name in message UI
  - Preference persisted in localStorage
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
- **Backend validation**: Model IDs validated against config before API calls
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

**Implementation Flow**:
```
Startup: backend/cmd/server/main.go loads config/models.json
  ↓
User opens Settings: Frontend fetches GET /api/models
  ↓
User selects model: Saved to localStorage.selectedModel
  ↓
User sends message: Model included in POST /api/chat/stream
  ↓
Backend validates model via config.IsValidModel(modelID)
  ↓
Backend passes model to llm.ChatWithHistoryStream(modelOverride)
  ↓
LLM uses provided model or defaults to first in config
  ↓
Backend sends model name via SSE: data: MODEL:model-name
  ↓
Frontend displays model in message: "AI (model-name)"
  ↓
Backend saves to DB: messages.model column
```

### Message History Pattern
- Full conversation history sent with every request to LLM for context coherence
- System prompt **always** prepended as first message: `{role: "system", content: "...prompt..."}`
- History retrieved in chronological order from DB

### Frontend State Management
- Minimal React state: messages, input, loading, conversationId, model, temperature, systemPrompt, responseFormat, responseSchema, conversationFormat, conversationSchema, theme, settingsOpen
- No Redux/complex state library; useContext for theme, useState for component local state
- localStorage for persistence: auth_token, theme, systemPrompt, responseFormat, responseSchema, selectedModel, temperature
- Component separation: Message.tsx handles format-specific rendering and displays model/temperature, SettingsModal.tsx handles model selection and temperature slider

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
  created_at TIMESTAMP
)
  ↓
conversation_summaries (
  id UUID PRIMARY KEY,
  conversation_id UUID REFERENCES conversations ON DELETE CASCADE,
  summary_content TEXT NOT NULL,                 -- LLM-generated summary of conversation
  summarized_up_to_message_id UUID REFERENCES messages ON DELETE SET NULL,  -- Last message included in summary
  usage_count INTEGER DEFAULT 0,                 -- How many times this summary was used (triggers re-summarization at 2+)
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

**Key Features:**
- All IDs are UUID type for distributed system compatibility and collision resistance
- Conversations auto-created on first message with first 100 chars as title
- **response_format** column locks format after first message (cannot be changed)
- **response_schema** stores JSON/XML schema definition for validation
- **model** column tracks which LLM model generated each assistant response
- **temperature** column tracks temperature setting (0.0-2.0) used for each response
- **conversation_summaries** table stores LLM-generated conversation summaries:
  - **summary_content**: Text summary of conversation up to a specific message
  - **summarized_up_to_message_id**: Tracks which message was the last one summarized
  - **usage_count**: Increments each time summary is used; triggers re-summarization at 2+
  - Multiple summaries per conversation supported (progressive re-summarization)
  - Most recent summary retrieved via `ORDER BY created_at DESC LIMIT 1`
- Messages store role ('user' or 'assistant') for history reconstruction
- Cascade deletes prevent orphaned records
- Indexes on user_id, conversation_id, and conversation_summaries.conversation_id for query performance
- UUIDs generated on the backend using `google/uuid` package
- COALESCE used in queries to handle NULL values: `COALESCE(response_format, 'text')`, `COALESCE(model, '')`

## Configuration

### Environment Variables (.env)

**Required:**
- `OPENROUTER_API_KEY` - API key for OpenRouter LLM service

**Optional (with defaults):**
- `OPENROUTER_SYSTEM_PROMPT` - Default system prompt (default: "You are a helpful assistant.")
- **Format-Aware LLM Parameters** (temperature is now user-controlled via Settings UI):
  - `OPENROUTER_TEXT_TOP_P` - Top-P for text conversations (default: 0.9)
  - `OPENROUTER_TEXT_TOP_K` - Top-K for text conversations (default: 40)
  - `OPENROUTER_STRUCTURED_TOP_P` - Top-P for JSON/XML (default: 0.8)
  - `OPENROUTER_STRUCTURED_TOP_K` - Top-K for JSON/XML (default: 20)
- `DB_HOST` - PostgreSQL host (default: postgres in Docker, localhost locally)
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_USER` - PostgreSQL user (default: postgres)
- `DB_PASSWORD` - PostgreSQL password (default: postgres)
- `DB_NAME` - Database name (default: chatapp)
- `DB_SSLMODE` - SSL mode (default: disable)
- `REACT_APP_API_URL` - Backend URL for frontend (default: http://localhost:8080)

**Note**: `OPENROUTER_MODEL` environment variable has been deprecated. Model selection is now managed via `backend/config/models.json`.

### Model Configuration (backend/config/models.json)

Available LLM models are configured in a JSON file with the following structure:

```json
[
  {
    "id": "meta-llama/llama-3.3-8b-instruct:free",
    "name": "Llama 3.3 8B Instruct (Free)",
    "provider": "Meta",
    "tier": "free"
  },
  {
    "id": "mistralai/mistral-7b-instruct:free",
    "name": "Mistral 7B Instruct (Free)",
    "provider": "Mistral AI",
    "tier": "free"
  },
  {
    "id": "z-ai/glm-4.5-air:free",
    "name": "GLM 4.5 Air (Free)",
    "provider": "Z-AI",
    "tier": "free"
  },
  {
    "id": "openrouter/polaris-alpha",
    "name": "Polaris Alpha",
    "provider": "OpenRouter",
    "tier": "paid"
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
  cmd/server/main.go              # Entry point, route setup (Go 1.22+ method-based routing), server start, models config loading
  config/models.json              # Available LLM models configuration
  internal/auth/auth.go           # JWT, login, register, middleware
  internal/config/models.go       # Models configuration loader and validator
  internal/db/                    # Database layer (postgres.go, user.go, conversation.go)
  internal/handlers/chat.go       # HTTP handlers for /api/chat and /api/models endpoints
  internal/llm/openrouter.go      # LLM integration, message building, streaming, model selection

frontend/
  src/App.tsx                     # Root, auth routing
  src/components/Chat.tsx         # Main chat UI, message streaming, model state management
  src/components/Message.tsx      # Message rendering with model display
  src/components/Login.tsx        # Auth forms
  src/components/SettingsModal.tsx # System prompt, model, and format configuration
  src/services/auth.ts            # JWT token management
  src/services/chat.ts            # API calls, SSE parsing, model fetching
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
- Default model: First model from `backend/config/models.json` (currently: meta-llama/llama-3.3-8b-instruct:free)
- Stream support: Full SSE streaming capability
- System prompt: First message with role='system'

**Model Selection:**
- **Configuration**: Models defined in `backend/config/models.json`
- **API Endpoint**: `GET /api/models` returns available models
- **Runtime Selection**: User chooses model from Settings UI dropdown
- **Override**: Optional `model` parameter in chat request overrides default
- **Validation**: Backend validates model ID via `config.IsValidModel()` before API calls
- **Tracking**: Model name saved in `messages.model` column for each response
- **Display**: Model name shown in UI next to "AI" label

**Available Models** (as configured):
1. **Llama 3.3 8B Instruct** (Meta, free): `meta-llama/llama-3.3-8b-instruct:free`
2. **Mistral 7B Instruct** (Mistral AI, free): `mistralai/mistral-7b-instruct:free`
3. **GLM 4.5 Air** (Z-AI, free): `z-ai/glm-4.5-air:free`
4. **Polaris Alpha** (OpenRouter, paid): `openrouter/polaris-alpha`

**Response Format (Streaming):**
```
data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"delta":{"content":" world"}}]}
data: MODEL:meta-llama/llama-3.3-8b-instruct:free
data: [DONE]
```

**Metadata Events** (SSE with prefixes):
- `CONV_ID:uuid-string` - Conversation ID for new conversations
- `MODEL:model-name` - Model used for generating response
- `TEMPERATURE:0.70` - Temperature used for generating response

Parsed by frontend: unescape newlines, collect chunks, capture metadata, skip [DONE]

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
  - `renderJsonAsTree()`: Parses JSON, displays as hierarchical tree supporting nested objects/arrays at unlimited depth with collapsible raw view
  - `renderXmlAsTree()`: Parses XML with DOMParser, displays as hierarchical tree with collapsible raw view
  - Uses `<details>/<summary>` HTML elements for collapsible raw views
  - Recursive rendering for unlimited nesting depth

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
// Temperature is now user-provided via Settings UI (0.0-2.0)
// Top-P and Top-K still from environment variables
func GetTopP(format string) *float64 {
  if format == "json" || format == "xml" {
    return os.Getenv("OPENROUTER_STRUCTURED_TOP_P")  // 0.8
  }
  return os.Getenv("OPENROUTER_TEXT_TOP_P")  // 0.9
}
```

### Visual Rendering

**JSON Format:**
```
[View Raw JSON ▼]  ← Collapsible details element

user: {...}
  ├─ name: "John Doe"
  ├─ age: 30
  ├─ active: true
  ├─ tags: [3 items]
  │   ├─ [0]: "developer"
  │   ├─ [1]: "golang"
  │   └─ [2]: "react"
  └─ address: {...}
      ├─ city: "New York"
      ├─ country: "USA"
      └─ coordinates: {...}
          ├─ lat: 40.7128
          └─ lng: -74.0060
```

**Rendering Features:**
- Unlimited nesting depth with 20px indentation per level
- Color-coded keys (primary color, bold) and values
- Type-aware rendering: strings with quotes, numbers/booleans plain
- Arrays show `[N items]` count with indexed elements `[0]`, `[1]`, etc.
- Objects show `{...}` indicator with nested properties
- Left border (3px solid) for structure clarity
- Alternating backgrounds for nested levels
- Handles null values with italic styling

**XML Format:**
```
[View Raw XML ▼]  ← Collapsible details element

<response version="1.0">
  ├─ <question>
  │   What is AI?
  └─ <answer>
      Artificial Intelligence
</response>

↑ Rendered as hierarchical tree with:
- Color-coded tags (primary color)
- Attribute display with proper formatting
- Left border (3px solid) for structure
- Alternating backgrounds for nesting levels
- Inline text for simple elements
- Full tree expansion for complex elements
```

### Key Features

1. **Format Locking**: Prevents format changes mid-conversation (data integrity)
2. **Schema Enforcement**: LLM instructed to strictly follow schema
3. **Visual Parsing**: JSON/XML parsed and rendered as hierarchical trees
4. **Unlimited Nesting**: Recursive rendering supports any depth of nested structures
5. **Raw View**: Collapsible access to original response
6. **Error Handling**: Falls back to `<pre>` if parsing fails
7. **Parameter Optimization**: Lower temperature for structured formats (0.3 vs 0.7)
8. **Type-Aware Display**: Different colors/styles for strings, numbers, booleans, null, objects, arrays

## Conversation Summarization Feature

### Overview
The application supports manual conversation summarization with progressive re-summarization. This allows long conversations to use summaries instead of full message history as LLM context, reducing token usage while maintaining conversation coherence.

### How It Works

**User Trigger:**
- User clicks 📝 button in chat interface
- Sends POST request to `/api/conversations/{id}/summarize`

**First Summarization:**
1. Backend checks if active summary exists via `db.GetActiveSummary(conversationID)`
2. If no summary: Retrieves all messages from conversation
3. Calls `llm.ChatForSummarization()` with summarization-only system prompt (no default system prompt)
4. Creates new summary in database with `db.CreateSummary()`
5. Returns summary content and `summarized_up_to_message_id`

**Progressive Re-Summarization (after 2+ uses):**
1. Backend detects `activeSummary.UsageCount >= 2`
2. Builds input: `[old summary as context] + [messages after last summarized message]`
3. Calls LLM to create new summary from combined content
4. Saves new summary to database (old summary remains for history)
5. Updates conversation's `active_summary_id` to new summary

**Usage in Chat:**
- When user sends new message, `ChatStreamHandler` checks for active summary
- If summary exists:
  - Fetches only messages AFTER `summarized_up_to_message_id`
  - Prepends summary as context to system prompt
  - Increments summary `usage_count`
- LLM receives: `[system prompt with summary context] + [recent messages only]`

### Frontend Implementation

**Components:**
- `Chat.tsx`:
  - State: `summaries: Array<{ upToMessageId: string; content: string }>`
  - `handleSummarize()`: Calls API, adds to summaries array, reloads messages to get IDs
  - Loads summaries on conversation open via `chatService.getConversationSummaries()`

- `chat.ts`:
  - `summarizeConversation()`: POST to `/api/conversations/{id}/summarize`
  - `getConversationSummaries()`: GET all summaries for displaying on conversation load

**Visual Display:**
```tsx
{messages.map((msg, idx) => {
  const summaryForThisMessage = summaries.find(s => s.upToMessageId === msg.id);
  return (
    <>
      <Message {...msg} />
      {summaryForThisMessage && (
        <details>
          <summary>📋 Messages above have been summarized (click to view)</summary>
          <div>{summaryForThisMessage.content}</div>
        </details>
      )}
    </>
  );
})}
```

### Backend Implementation

**Database Functions (`internal/db/conversation.go`):**
- `CreateSummary(conversationID, summaryContent, summarizedUpToMessageID)`: Creates new summary
- `GetActiveSummary(conversationID)`: Returns most recent summary (`ORDER BY created_at DESC LIMIT 1`)
- `GetAllSummaries(conversationID)`: Returns all summaries in chronological order
- `IncrementSummaryUsageCount(summaryID)`: Increments usage_count by 1
- `GetMessagesAfterMessage(conversationID, afterMessageID)`: Fetches messages created after specific message
- `GetLastMessageID(conversationID)`: Returns ID of most recent message
- `UpdateConversationActiveSummary(conversationID, summaryID)`: Updates active_summary_id reference

**API Endpoints (`internal/handlers/chat.go`):**
- `SummarizeConversationHandler`:
  - POST `/api/conversations/{id}/summarize`
  - Validates ownership, determines summarization strategy, calls LLM, saves result
  - Returns: `{summary, summarized_up_to_message_id, conversation_id}`

- `GetConversationSummariesHandler`:
  - GET `/api/conversations/{id}/summaries`
  - Returns all summaries for a conversation
  - Response: `{summaries: [{id, summary_content, summarized_up_to_message_id, usage_count, created_at}]}`

**LLM Integration (`internal/llm/openrouter.go`):**
- `ChatForSummarization(messages, summarizationPrompt, model, temperature)`:
  - Uses `buildMessagesWithCustomSystemPrompt()` instead of `buildMessagesWithHistory()`
  - Sends ONLY the summarization prompt (no default system prompt)
  - Returns plain text summary

**Summarization Logic:**
```go
// Check existing summary
activeSummary, err := db.GetActiveSummary(convID)

if err != nil || activeSummary == nil {
    // No summary - summarize all messages
    messagesToSummarize, _ = db.GetConversationMessages(convID)
} else if activeSummary.UsageCount >= 2 {
    // Re-summarize: old summary + new messages
    messagesToSummarize = []llm.Message{
        {Role: "assistant", Content: fmt.Sprintf("Previous summary:\n%s", activeSummary.SummaryContent)},
    }
    newMessages, _ := db.GetMessagesAfterMessage(convID, *activeSummary.SummarizedUpToMessageID)
    messagesToSummarize = append(messagesToSummarize, newMessages...)
} else {
    // Summary exists but usage_count < 2 - return existing
    return existingSummary
}
```

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
- **Most recent summary**: Retrieved via `ORDER BY created_at DESC LIMIT 1` (not via `active_summary_id`)
- **Usage tracking**: `usage_count` incremented on each chat message sent after summary exists
- **Message IDs required**: Frontend reloads messages after summarization to get server-generated UUIDs
- **Collapsible UI**: Uses HTML5 `<details>` element for expandable summary view
- **Persistence**: All summaries stored in database, loaded on conversation open
