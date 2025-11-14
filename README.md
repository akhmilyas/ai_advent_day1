# AI Chat Application

A fullstack chat application with Go backend, React frontend, PostgreSQL database, and OpenRouter LLM integration.

## Features

- 🤖 **Multi-Model Support** - 10+ LLM models (Meta, Google, OpenAI, Anthropic, etc.)
- 🎛️ **Temperature Control** - Adjustable creativity (0.0-2.0)
- 📊 **Structured Formats** - JSON/XML with schema validation and tree rendering
- 📝 **Smart Summarization** - Progressive re-summarization for long conversations
- ⚡ **Real-time Streaming** - Server-Sent Events (SSE)
- 🎨 **Dark/Light Theme** - Persistent theme preference
- 🔐 **JWT Authentication** - Secure user auth with bcrypt

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

```
Frontend (React, Port 3000)
    ↓ HTTP REST + SSE
Backend (Go, Port 8080)
    ↓ SQL
PostgreSQL (Port 5432)
    ↑ HTTPS
OpenRouter LLM API
```

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

```bash
# Required
OPENROUTER_API_KEY=your_api_key

# Optional LLM
OPENROUTER_SYSTEM_PROMPT=You are a helpful assistant.

# LLM Parameters (Temperature is user-controlled via UI)
OPENROUTER_TEXT_TOP_P=0.9          # Text conversations
OPENROUTER_TEXT_TOP_K=40
OPENROUTER_STRUCTURED_TOP_P=0.8    # JSON/XML (more deterministic)
OPENROUTER_STRUCTURED_TOP_K=20

# Database (Docker defaults)
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=chatapp
DB_SSLMODE=disable
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

First model is the default. Currently configured:
- Llama 3.3 8B (Meta, free) - **DEFAULT**
- Mistral 7B (Mistral, free)
- GLM 4.5 Air (Z-AI, free)
- Polaris Alpha (OpenRouter, free)
- Gemini 2.5 Flash (Google, paid)
- Claude Sonnet 4.5 (Anthropic, paid)
- And 4 more...

## Usage

1. **Login/Register** - Use demo credentials or create account
2. **Configure Settings** (⚙️):
   - Select model from dropdown
   - Adjust temperature (0.0-2.0 slider)
   - Choose response format (Text/JSON/XML)
   - Add schema (for JSON/XML)
   - Set custom system prompt (for Text)
3. **Chat** - Type message, AI streams response
4. **Summarize** (📝) - Click to create conversation summary
5. **Theme** - Toggle 🌙/☀️ for dark/light mode

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

**Backend**: Go 1.25.3, PostgreSQL 15, JWT, UUID, Bcrypt
**Frontend**: React 18, TypeScript, react-markdown, remark-gfm
**Deployment**: Docker, Docker Compose, nginx
**LLM**: OpenRouter API (10+ models)

## Database Schema

```sql
users (id UUID, username, email, password_hash, created_at)
  ↓
conversations (id UUID, user_id, title, response_format, response_schema,
               active_summary_id, created_at, updated_at)
  ↓
messages (id UUID, conversation_id, role, content, model, temperature,
          prompt_tokens, completion_tokens, total_tokens, total_cost,
          latency, generation_time, provider, created_at)
  ↓
conversation_summaries (id UUID, conversation_id, summary_content,
                        summarized_up_to_message_id, usage_count, created_at)
```

## Project Structure

```
backend/
  cmd/server/main.go              # Entry point, routing
  config/models.json              # Model configuration
  internal/
    auth/auth.go                  # JWT authentication
    config/models.go              # Config loader
    db/                           # Database layer
    handlers/chat.go              # HTTP handlers
    llm/                          # OpenRouter integration

frontend/
  src/
    components/
      Chat.tsx                    # Main UI
      Message.tsx                 # Format rendering
      SettingsModal.tsx           # Configuration
      Sidebar.tsx                 # Conversations
    services/
      auth.ts                     # JWT management
      chat.ts                     # API calls
    contexts/ThemeContext.tsx     # Theme state

docker-compose.yml                # Orchestration
.env.example                      # Config template
CLAUDE.md                         # Developer docs
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
| OPENROUTER_API_KEY not set | Add to `.env` and restart |
| Port already in use | Change ports in `docker-compose.yml` |
| Database connection error | Check `DB_*` vars in `.env` |
| Login fails | Use `demo`/`demo123` or register |
| Stream stops | Check network, browser console |
| Settings not saving | Enable localStorage in browser |
| CORS errors | Verify backend at http://localhost:8080/api/health |
| Format locked | Start new conversation to change format |

**Debug**:
```bash
# Backend logs
docker compose logs backend -f

# Database access
docker exec -it <container> psql -U postgres -d chatapp
```

## Development

### Add API Endpoint
1. Add handler to `backend/internal/handlers/chat.go`
2. Register route in `backend/cmd/server/main.go`
3. Add method to `frontend/src/services/chat.ts`
4. Use in component

### Add Model
1. Edit `backend/config/models.json`
2. Restart backend
3. Model appears in Settings dropdown

### Testing
```bash
# Frontend
cd frontend && npm test

# Backend
cd backend && go test ./...
```

## License

MIT

---

**For detailed documentation, see [CLAUDE.md](CLAUDE.md)**
