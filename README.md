# AI Chat Application

A fullstack chat app with Go backend, React frontend, PostgreSQL, and OpenRouter LLM API integration.

**Features**: User auth (JWT), conversation history, SSE streaming, dark/light theme, markdown rendering, customizable system prompts, **model selection**, **temperature control**, **structured response formats (JSON/XML)** with visual rendering

## Quick Start

### Prerequisites
- Docker & Docker Compose (recommended), or Go 1.25.3+, Node.js 20+, PostgreSQL 13+
- OpenRouter API key from [openrouter.ai](https://openrouter.ai/)

### Setup
```bash
# Copy environment file
cp .env.example .env

# Edit .env with your OpenRouter API key
OPENROUTER_API_KEY=your_key_here

# Build and run
docker compose build
docker compose up
```

Access at `http://localhost:3000` (frontend) and `http://localhost:8080` (backend)

**Demo credentials**: `demo` / `demo123`

## Architecture

```
Frontend (React, Port 3000)
    ↓ HTTP REST + SSE
Backend (Go, Port 8080)
    ↓ SQL
PostgreSQL (Port 5432)
    ↑ HTTPS API
OpenRouter LLM (External)
```

### Key Components

**Backend**: Auth (JWT), Chat handlers (REST + SSE), LLM service, Database layer
**Frontend**: Login/Register, Chat UI, Auth service, Chat service, Theme system
**Database**: users, conversations, messages tables

## API Endpoints

### Public
- `POST /api/login` → `{username, password}` → `{token}`
- `POST /api/register` → `{username, email, password}` → `{token}`
- `GET /api/health` → OK

### Protected (require `Authorization: Bearer <token>`)
- `GET /api/models` → `{models: [{id, name, provider, tier}, ...]}`
- `POST /api/chat` → `{message, conversation_id?, system_prompt?, response_format?, response_schema?, model?, temperature?}` → `{response, conversation_id, model}`
- `POST /api/chat/stream` → `{message, conversation_id?, system_prompt?, response_format?, response_schema?, model?, temperature?}` → SSE stream
- `GET /api/conversations` → `{conversations: [{id, title, response_format, response_schema, ...}, ...]}`
- `GET /api/conversations/{id}/messages` → `{messages: [{role, content, model, temperature, ...}, ...]}`
- `DELETE /api/conversations/{id}` → `{success: boolean}`

**CORS**: All endpoints support Cross-Origin requests from any origin (frontend can call backend from browser)

**Response Formats**:
- `text` (default): Plain text with markdown rendering
- `json`: Structured JSON with schema validation, rendered as hierarchical tree
- `xml`: Structured XML with schema validation, rendered as hierarchical tree

**Note**: Response format is locked after the first message in a conversation and stored in the database

## Build & Run

### With Docker Compose
```bash
docker compose build
docker compose up
```

### Manual Build

**Backend**:
```bash
cd backend
go mod download
go build -o server ./cmd/server
./server
```

**Frontend**:
```bash
cd frontend
npm install
npm run build
# or for dev: npm start
```

## Configuration

### Environment Variables

Set environment variables in `.env`:

```bash
# Required
OPENROUTER_API_KEY=your_api_key

# Optional LLM
OPENROUTER_SYSTEM_PROMPT=You are a helpful assistant.

# LLM Parameters - Format-Aware Configuration
# Note: Temperature is now user-controlled via Settings UI (0.0-2.0 slider)
# Parameters for plain text conversations
OPENROUTER_TEXT_TOP_P=0.9
OPENROUTER_TEXT_TOP_K=40

# Parameters for structured formats: JSON/XML (more deterministic)
OPENROUTER_STRUCTURED_TOP_P=0.8
OPENROUTER_STRUCTURED_TOP_K=20

# Optional Database (Docker defaults shown)
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=chatapp
DB_SSLMODE=disable
```

### Model Configuration

Available models are configured in `backend/config/models.json`:

```json
[
  {
    "id": "meta-llama/llama-3.3-8b-instruct:free",
    "name": "Llama 3.3 8B Instruct (Free)",
    "provider": "Meta",
    "tier": "free"
  },
  {
    "id": "google/gemini-2.5-flash",
    "name": "Gemini 2.5 Flash",
    "provider": "Google",
    "tier": "paid"
  },
  {
    "id": "openai/gpt-5-mini",
    "name": "GPT-5 Mini",
    "provider": "OpenAI",
    "tier": "paid"
  },
  {
    "id": "z-ai/glm-4.5-air:free",
    "name": "GLM 4.5 Air (Free)",
    "provider": "Z-AI",
    "tier": "free"
  },
  {
    "id": "alibaba/tongyi-deepresearch-30b-a3b:free",
    "name": "Tongyi DeepResearch 30B (Free)",
    "provider": "Alibaba",
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

**Note**: The first model in the configuration file is used as the default. Users can select a different model from the Settings UI.

## Usage

1. **Register/Login**: Create account or use `demo/demo123`
2. **Start New Chat**: Click sidebar or start typing
3. **Configure Settings** (⚙️):
   - **Select Model**: Choose from available models (Llama 3.3, Gemini 2.5 Flash, GPT-5 Mini, GLM 4.5 Air, Tongyi DeepResearch, Polaris Alpha)
   - **Temperature**: Adjust creativity (0.0-2.0 slider, default 0.7)
     - 0.0-0.5: Focused and deterministic
     - 0.5-1.0: Balanced
     - 1.0-2.0: Creative and random
   - **Response Format** (before first message):
     - **Plain Text**: Natural conversation with markdown rendering
     - **JSON**: Structured data with schema, displayed as hierarchical tree with raw view
     - **XML**: Structured markup with schema, displayed as hierarchical tree with raw view
   - **System Prompt** (text mode only): Customize AI behavior
   - **Schema** (JSON/XML only): Define structure before first message
4. **Chat**: Type message → AI streams response in real-time
5. **Theme**: Toggle 🌙/☀️ button for dark/light mode
6. **Conversations**: Auto-saved with format locked after first message
7. **Model & Temperature Display**: Each AI response shows which model and temperature were used
8. **Logout**: Click logout button (all data persisted)

## Tech Stack

**Backend**: Go 1.25.3, PostgreSQL 13, jwt-go, bcrypt, google/uuid
**Frontend**: React 18, TypeScript, react-markdown, remark-gfm
**Deployment**: Docker, Docker Compose

**IDs**: All database IDs use UUID (Universally Unique Identifiers) for better distributed system support and collision resistance

## Features

- **Auth**: JWT tokens (24hr), bcrypt password hashing, user registration
- **Chat**: SSE streaming, optimistic UI updates, full conversation history
- **Model Selection**:
  - Choose from multiple LLM models (configured via `backend/config/models.json`)
  - Current models: Llama 3.3 8B (free), Gemini 2.5 Flash, GPT-5 Mini, GLM 4.5 Air, Tongyi DeepResearch, Polaris Alpha
  - Default model auto-selected from configuration
  - Model preference saved to localStorage
  - Per-message model tracking in database
  - Model name displayed with each AI response
- **Temperature Control**:
  - User-adjustable temperature slider (0.0-2.0, step 0.01)
  - Default: 0.7 (balanced)
  - Per-request temperature sent to OpenRouter API
  - Temperature saved with each message in database
  - Temperature displayed with each AI response
  - Preference persisted in localStorage
- **Response Formats**:
  - **Text**: Markdown rendering with tables, lists, code blocks
  - **JSON**: Schema-based structured output, rendered as hierarchical tree supporting nested objects/arrays with raw view toggle
  - **XML**: Schema-based structured output, rendered as hierarchical tree with syntax highlighting and raw view toggle
- **Format-Aware LLM Parameters**: Different top-p/top-k for text vs structured formats
- **OpenRouter Provider Routing**: `require_parameters: true` ensures all parameters are supported by selected provider
- **System Prompts**: Custom prompts for text conversations (stored in localStorage)
- **Schema Validation**: Define JSON/XML schemas for structured responses
- **Visual Rendering**: Hierarchical tree structures for both JSON and XML with unlimited nesting support
- **Format Locking**: Response format cannot be changed after conversation starts
- **Database**: PostgreSQL persistence with format/schema/temperature stored per conversation/message
- **Security**: JWT validation, CORS, API key management

## Project Structure

```
backend/
  cmd/server/main.go           # Entry point, routing
  config/models.json           # Available LLM models configuration
  internal/auth/               # JWT, login, register
  internal/config/             # Models configuration loader
  internal/db/                 # PostgreSQL layer (users, conversations, messages)
  internal/handlers/           # HTTP handlers (chat, conversations, models)
  internal/llm/                # OpenRouter integration, format-aware params
frontend/
  src/components/
    Chat.tsx                   # Main chat UI, model selection
    Message.tsx                # Message rendering (text/JSON/XML)
    SettingsModal.tsx          # Format/schema/prompt/model configuration
    Sidebar.tsx                # Conversation list
  src/services/
    auth.ts                    # JWT token management
    chat.ts                    # API calls, SSE parsing, model fetching
  src/contexts/
    ThemeContext.tsx           # Dark/light theme
  src/themes.ts                # Color palettes
docker-compose.yml             # Service orchestration
.env.example                   # Configuration template
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| OPENROUTER_API_KEY not set | Add to `.env` |
| Port already in use | Change in `docker-compose.yml` |
| Database connection error | Check DB_HOST, DB_PORT, credentials in `.env` |
| Login fails | Use demo/demo123 or register new account |
| Stream stops | Check network connection, browser console for errors |
| Settings/theme/prompt not saving | Enable localStorage in browser |
| CORS errors | Verify backend is running and accessible |

## License

MIT
