# AI Chat Application

A fullstack chat app with Go backend, React frontend, PostgreSQL, and OpenRouter LLM API integration.

**Features**: User auth (JWT), conversation history, SSE streaming, dark/light theme, markdown rendering, customizable system prompts

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
- `POST /api/chat` → `{message, conversation_id?, system_prompt?}` → `{response, conversation_id, model}`
- `POST /api/chat/stream` → `{message, conversation_id?, system_prompt?}` → SSE stream
- `GET /api/conversations` → `{conversations: [...]}`

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

Set environment variables in `.env`:

```bash
# Required
OPENROUTER_API_KEY=your_api_key

# Optional LLM
OPENROUTER_MODEL=meta-llama/llama-3.3-8b-instruct:free
OPENROUTER_SYSTEM_PROMPT=You are a helpful assistant.

# Optional Database (Docker defaults shown)
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=chatapp
DB_SSLMODE=disable
```

## Usage

1. **Register/Login**: Create account or use `demo/demo123`
2. **Chat**: Type message → AI streams response in real-time
3. **System Prompt**: Click ⚙️ Settings button to customize AI behavior (saved locally)
4. **Theme**: Toggle 🌙/☀️ button for dark/light mode
5. **Conversations**: Auto-created and persisted in database
6. **Logout**: Click logout button (conversations saved)

## Tech Stack

**Backend**: Go 1.25.3, PostgreSQL 13, jwt-go, bcrypt
**Frontend**: React 18, TypeScript, react-markdown, remark-gfm
**Deployment**: Docker, Docker Compose

## Features

- **Auth**: JWT tokens (24hr), bcrypt password hashing, user registration
- **Chat**: SSE streaming, optimistic UI updates, full conversation history
- **System Prompts**: Global session-wide custom prompts merged with env default, localStorage persistence
- **Markdown**: Tables, lists, headers, code blocks with syntax highlighting
- **Database**: PostgreSQL persistence for users/conversations/messages
- **Security**: JWT validation, CORS, API key management

## Project Structure

```
backend/
  cmd/server/main.go
  internal/auth/
  internal/db/
  internal/handlers/
  internal/llm/
frontend/
  src/components/
  src/services/
  src/contexts/
docker-compose.yml
.env.example
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
