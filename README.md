# AI Chat Application

A fullstack chat application with Go backend, React frontend, PostgreSQL database, and dual LLM provider support (OpenRouter + Genkit).

## Features

- 🤖 **Multi-Model Support** - 10+ LLM models (Meta, Google, OpenAI, Z-AI, Alibaba, etc.)
- 🎛️ **Temperature Control** - Adjustable creativity (0.0-2.0 slider)
- 📊 **Structured Formats** - JSON/XML with schema validation and tree rendering
- 📝 **Smart Summarization** - Progressive re-summarization for long conversations
- ⚡ **Real-time Streaming** - Server-Sent Events (SSE)
- 🎨 **Dark/Light Theme** - Persistent theme preference
- 🔐 **JWT Authentication** - Secure user auth with bcrypt
- 💰 **Cost Tracking** - Token usage and cost metrics per message
- 🔀 **Dual Providers** - OpenRouter (production) + Genkit (experimental)
- 📖 **War and Peace Testing** - Context window testing with large text injection

## Quick Start

### Prerequisites
- Docker & Docker Compose (recommended)
- Or: Go 1.25.3+, Node.js 20+, PostgreSQL 13+
- OpenRouter API key from [openrouter.ai](https://openrouter.ai/)

### Setup
```bash
# Copy environment file
cp .env.example .env

# Edit .env with your configuration
# Required:
OPENROUTER_API_KEY=your_key_here
JWT_SECRET=your-secure-32-char-minimum-secret

# Generate a secure JWT secret (32+ characters)
# openssl rand -base64 32

# Build and run
docker compose build
docker compose up
```

Access at:
- Frontend: http://localhost:3000
- Backend: http://localhost:8080

**Demo credentials**: `demo` / `demo123`

## Architecture

### High-Level
```
Frontend (React/TypeScript, Port 3000)
    ↓ HTTP REST + SSE
Backend (Go, Port 8080) - Clean Layered Architecture
    ├─ API Layer (Handlers)
    ├─ Service Layer (Business Logic)
    ├─ Repository Layer (Database)
    └─ Validation Layer
    ↓ SQL
PostgreSQL (Port 5432) with golang-migrate
    ↑ HTTPS
OpenRouter / Genkit LLM APIs
```

### Backend Layers
- **API Handlers** (`internal/api/handlers/`) - HTTP request/response, SSE streaming
- **Service Layer** (`internal/service/`) - Chat, conversations, summaries, LLM providers
- **Repository** (`internal/repository/`) - Database interface abstraction (PostgreSQL)
- **Validation** (`pkg/validation/`) - Request validators with comprehensive tests
- **Config** (`internal/config/`) - Centralized environment + JSON configuration

## API Endpoints

### Public
- `POST /api/login` - Authenticate user
- `POST /api/register` - Create account
- `GET /api/health` - Health check

### Protected (require `Authorization: Bearer <token>`)
- `GET /api/models` - List available models
- `POST /api/chat/stream` - Send message (SSE streaming)
- `POST /api/chat` - Send message (blocking)
- `GET /api/conversations` - List conversations
- `GET /api/conversations/{id}/messages` - Get messages
- `DELETE /api/conversations/{id}` - Delete conversation
- `POST /api/conversations/{id}/summarize` - Create/update summary
- `GET /api/conversations/{id}/summaries` - List summaries

## Configuration

### Environment Variables (.env)

**Required:**
```bash
OPENROUTER_API_KEY=your_api_key
JWT_SECRET=your-secure-32-char-minimum-secret  # MUST be 32+ characters
```

**Optional (with defaults):**
```bash
# Server
SERVER_PORT=8080

# LLM Configuration
OPENROUTER_SYSTEM_PROMPT=You are a helpful assistant.

# Format-Aware LLM Parameters (Temperature is user-controlled via UI)
OPENROUTER_TEXT_TOP_P=0.9          # Text conversations
OPENROUTER_TEXT_TOP_K=40
OPENROUTER_STRUCTURED_TOP_P=0.8    # JSON/XML (more deterministic)
OPENROUTER_STRUCTURED_TOP_K=20

# Authentication
JWT_EXPIRATION_HOURS=24

# Database (Docker defaults)
DB_HOST=postgres               # Use 'localhost' for manual runs
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=chatapp
DB_SSLMODE=disable

# Frontend
REACT_APP_API_URL=http://localhost:8080
```

### Model Configuration

Edit `backend/config/models.json` to add/remove models:

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

**Default Model:** First model in the array is used as the default.

**Currently configured models:**
- Llama 3.3 8B (Meta, free) - **DEFAULT**
- Gemini 2.5 Flash (Google, paid)
- GPT-5 Mini (OpenAI, paid)
- GLM 4.5 Air (Z-AI, free)
- Tongyi DeepResearch 30B (Alibaba, free)
- Polaris Alpha (OpenRouter, paid)
- Plus 4 additional models (see `config/models.json`)

## Usage

1. **Login/Register** - Use demo credentials (`demo`/`demo123`) or create account
2. **Configure Settings** (⚙️):
   - Select LLM model from dropdown (10+ models available)
   - Adjust temperature (0.0-2.0 slider, default 0.7)
   - Choose LLM provider (OpenRouter or Genkit)
   - Choose response format (Text/JSON/XML) - *locks after first message*
   - Add schema (for JSON/XML formats)
   - Set custom system prompt (for Text format)
   - Enable War and Peace context injection (for testing large context windows)
3. **Chat** - Type message, AI streams response with real-time display
4. **Summarize** (📝) - Click to create/update conversation summary (progressive re-summarization)
5. **Theme** - Toggle 🌙/☀️ for dark/light mode
6. **Conversations** - View history, switch between conversations, delete old ones

### Response Formats

**Text** (default): Markdown rendering, custom prompts

**JSON**: Structured data with schema
```json
{
  "type": "object",
  "properties": {
    "answer": {"type": "string"}
  }
}
```

**XML**: Structured markup with schema
```xml
<xsd:schema>
  <xsd:element name="response" type="xsd:string"/>
</xsd:schema>
```

**Note**: Format locks after first message and cannot be changed.

## Tech Stack

**Backend:** Go 1.25.3, PostgreSQL 15, golang-migrate, JWT, UUID, Bcrypt
**Frontend:** React 18, TypeScript, react-markdown, remark-gfm
**Deployment:** Docker, Docker Compose, nginx
**LLM Providers:** OpenRouter API (production), Firebase Genkit (experimental)
**Testing:** Jest, React Testing Library, Go testing framework

## Database Schema

**Managed via golang-migrate** (7 migrations applied automatically on startup)

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
  response_format VARCHAR(10) DEFAULT 'text',    -- 'text', 'json', 'xml' (locked after first message)
  response_schema TEXT,                          -- Schema for JSON/XML
  active_summary_id UUID,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)
  ↓
messages (
  id UUID PRIMARY KEY,
  conversation_id UUID REFERENCES conversations,
  role VARCHAR,
  content TEXT,
  model VARCHAR(255),                            -- LLM model used
  temperature REAL,                              -- Temperature (0.0-2.0)
  provider VARCHAR(50),                          -- 'openrouter' or 'genkit'
  prompt_tokens INTEGER,                         -- Cost tracking
  completion_tokens INTEGER,
  total_cost DECIMAL(10, 6),
  latency INTEGER,                               -- Performance metrics (ms)
  generation_time INTEGER,
  war_and_peace_used BOOLEAN DEFAULT FALSE,      -- Context testing
  war_and_peace_percent INTEGER,
  created_at TIMESTAMP
)
  ↓
conversation_summaries (
  id UUID PRIMARY KEY,
  conversation_id UUID REFERENCES conversations ON DELETE CASCADE,
  summary_content TEXT NOT NULL,
  summarized_up_to_message_id UUID REFERENCES messages ON DELETE SET NULL,
  usage_count INTEGER DEFAULT 0,                 -- Triggers re-summarization at 2+
  created_at TIMESTAMP
)
  ↓
schema_migrations (
  version BIGINT PRIMARY KEY,                    -- Migration tracking
  dirty BOOLEAN NOT NULL
)
```

**Key Features:**
- All IDs are UUIDs for distributed compatibility
- Cascade deletes prevent orphaned records
- Indexes on foreign keys and frequently queried fields
- Cost and performance tracking per message
- Provider tracking (OpenRouter vs Genkit)

## Project Structure

```
backend/
  cmd/server/main.go              # Entry point, routing, config loading
  config/models.json              # LLM model configuration
  migrations/                     # Database migrations (golang-migrate)
    000001_create_users_table.up.sql / .down.sql
    000002_create_conversations_table.up.sql / .down.sql
    ... (7 total migrations)
  internal/
    api/handlers/                 # HTTP request handlers
      auth_handlers.go            # Login, register
      chat_handlers.go            # Chat, conversations, summarization
      models_handlers.go          # Model configuration
    service/                      # Business logic layer
      chat_service.go             # Chat logic, streaming
      conversation_service.go     # CRUD operations
      summary_service.go          # Summarization strategies
      openrouter_provider.go      # OpenRouter integration
      genkit_provider.go          # Genkit integration
    repository/                   # Database abstraction
      repository.go               # Interface definitions
      postgres_repository.go      # PostgreSQL implementation
    config/                       # Configuration loading
      app.go                      # AppConfig from environment
      models.go                   # Model config loader
    context/                      # Utilities
      warandpeace.go              # War and Peace text loader
    auth/                         # Authentication
      auth.go                     # JWT generation/validation
      middleware.go               # Auth middleware
  pkg/validation/                 # Request validators
    auth_validator.go             # Username, email, password
    chat_validator.go             # Message, temperature, format
    *_test.go                     # Comprehensive test coverage

frontend/
  src/
    components/
      Chat.tsx                    # Main chat UI
      Message.tsx                 # Format-specific rendering
      Login.tsx                   # Auth forms
      SettingsModal.tsx           # Settings UI (model, temp, format, provider)
      Sidebar.tsx                 # Conversation history
    services/
      auth.ts                     # JWT token management
      chat.ts                     # API calls, SSE parsing
    contexts/
      ThemeContext.tsx            # Theme state provider
    themes.ts                     # Color palettes
    App.tsx                       # Root component

docker-compose.yml                # Service orchestration
.env.example                      # Configuration template
CLAUDE.md                         # Developer documentation
README.md                         # This file (user documentation)
```

## Manual Build

**Backend**:
```bash
cd backend
go mod download
go build -o server ./cmd/server
./server  # Requires PostgreSQL on localhost:5432
```

**Frontend**:
```bash
cd frontend
npm install
npm run build    # Production
npm start        # Development
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| OPENROUTER_API_KEY not set | Add to `.env` and restart containers |
| JWT_SECRET validation error | Ensure JWT_SECRET is 32+ characters in `.env` |
| Port already in use | Change ports in `docker-compose.yml` |
| Database connection error | Check `DB_*` vars in `.env` (use `DB_HOST=postgres` for Docker) |
| Migration errors | Check `docker compose logs backend` and `schema_migrations` table |
| Login fails | Use `demo`/`demo123` or register new account |
| Stream stops mid-response | Check network tab, browser console for SSE errors |
| Settings not saving | Enable localStorage in browser, check console |
| CORS errors | Verify backend at http://localhost:8080/api/health |
| Format locked | Response format locks after first message - start new conversation |
| Model validation error | Verify model ID exists in `backend/config/models.json` |

**Debug Commands:**
```bash
# Backend logs (follow mode)
docker compose logs backend --follow

# Database access
docker compose exec postgres psql -U postgres -d chatapp

# Check migrations
docker compose exec postgres psql -U postgres -d chatapp -c "SELECT * FROM schema_migrations;"

# Restart services
docker compose restart backend
docker compose restart frontend

# Clean rebuild (removes volumes)
docker compose down -v
docker compose up --build
```

## Development

### Common Tasks

**Add API Endpoint:**
1. Add handler to `internal/api/handlers/` (auth, chat, or models handlers)
2. Add business logic to `internal/service/` layer
3. Add database operations to `internal/repository/` (interface + implementation)
4. Register route in `cmd/server/main.go` (both POST and OPTIONS for CORS)
5. Update `frontend/src/services/chat.ts` with new method
6. Use in React components

**Add Database Migration:**
```bash
cd backend
migrate create -ext sql -dir migrations -seq add_new_feature
# Edit .up.sql and .down.sql
docker compose restart backend  # Auto-applies migration
```

**Add LLM Model:**
1. Edit `backend/config/models.json` (add model ID, name, provider, tier)
2. Restart backend: `docker compose restart backend`
3. Model appears in Settings dropdown

**Add LLM Provider:**
1. Create provider file in `internal/service/` (implement ChatWithHistoryStream)
2. Add to ChatService struct
3. Update switch statement in `chat_handlers.go`

### Testing

```bash
# Frontend tests
cd frontend && npm test
cd frontend && CI=true npm test  # CI mode

# Backend tests (all packages)
cd backend && go test ./...

# Backend tests (specific package with verbose output)
cd backend && go test ./pkg/validation -v
cd backend && go test ./internal/service -v
```

**Test Coverage:**
- Validation layer: Comprehensive coverage in `pkg/validation/*_test.go`
- Frontend: Component tests with Jest + React Testing Library
- Backend: Go testing framework (no mocking library currently)

## Key Technical Features

### Clean Layered Architecture
- **API Layer**: HTTP handlers, auth middleware, SSE streaming
- **Service Layer**: Business logic, LLM provider strategies
- **Repository Layer**: Database abstraction (interface-based, PostgreSQL implementation)
- **Validation Layer**: Request validators with comprehensive test coverage
- **Config Layer**: Centralized environment + JSON configuration

### Database Migrations
- **golang-migrate** for version-controlled schema changes
- Auto-applied on backend startup
- 7 migrations tracking: users, conversations, messages, summaries, cost tracking, provider tracking, War and Peace fields
- Bidirectional support (up/down) for rollbacks

### LLM Provider Strategy Pattern
- **OpenRouter Provider** (production): Direct API, cost tracking via generation IDs, async fetching with retry
- **Genkit Provider** (experimental): Firebase Genkit, OpenAI-compatible layer
- Easily extensible for new providers

### Response Format System
- **Format locking**: Once conversation starts, format cannot change (stored in DB)
- **Text format**: Markdown rendering with custom prompts
- **JSON format**: Tree rendering with collapsible nodes + raw view
- **XML format**: Syntax-highlighted tree + raw view
- Schema enforcement for structured formats

### Cost and Performance Tracking
- Token counts (prompt + completion)
- Total cost in USD (async fetching from OpenRouter API)
- Latency (time to first token)
- Generation time (total streaming time)
- All metrics stored per message

### Conversation Summarization
- Manual trigger via 📝 button
- Progressive re-summarization after 2+ uses
- Reduces context size for long conversations
- Multiple summaries stored for history

## Security Notes

**Current Implementation:**
- JWT authentication with 32+ character secret (validated on startup)
- Bcrypt password hashing
- User ownership verification for conversations
- SQL injection protection via parameterized queries
- CORS enabled (allows all origins)

**Production Improvements Needed:**
- Move JWT secret to secrets manager (Vault, AWS Secrets Manager)
- Restrict CORS to specific frontend origin
- Add rate limiting on auth endpoints
- Enable HTTPS/TLS
- Implement refresh tokens
- Add request logging and monitoring

## License

MIT

---

**For detailed developer documentation, see [CLAUDE.md](CLAUDE.md)**
