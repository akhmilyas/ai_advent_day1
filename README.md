# AI Chat Application

A fullstack chat application with Go backend and React TypeScript frontend, featuring REST API integration with OpenRouter's LLM APIs and server-side conversation history management.

## Dependencies

### Backend
- **Go**: 1.21 or higher
- **Dependencies** (managed via go.mod):
  - `github.com/golang-jwt/jwt/v5` v5.2.0 - JWT authentication

### Frontend
- **Node.js**: 20.x or higher
- **npm**: 10.x or higher
- **Dependencies** (managed via package.json):
  - React 18.2.0
  - TypeScript 5.3.3
  - react-scripts 5.0.1

### Docker
- **Docker**: 20.10 or higher
- **Docker Compose**: 2.0 or higher

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  React TypeScript Application (Port 3000)            │  │
│  │  - Login Component (JWT Authentication)              │  │
│  │  - Chat Component (Message UI)                       │  │
│  │  - Auth Service (Token Management)                   │  │
│  │  - Chat Service (HTTP REST)                          │  │
│  │  - Theme System (Light/Dark modes)                   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                          │
                          │ HTTP REST API
                          │ JWT Bearer Token Authorization
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                         Backend                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Go HTTP Server (Port 8080)                          │  │
│  │  ┌────────────────────────────────────────────────┐ │  │
│  │  │  Routes:                                        │ │  │
│  │  │  POST /api/login (public)                      │ │  │
│  │  │  GET  /api/health (public)                     │ │  │
│  │  │  POST /api/chat (protected, REST)              │ │  │
│  │  └────────────────────────────────────────────────┘ │  │
│  │  ┌────────────────────────────────────────────────┐ │  │
│  │  │  Middleware:                                    │ │  │
│  │  │  - CORS Handler                                 │ │  │
│  │  │  - JWT Authentication                           │ │  │
│  │  └────────────────────────────────────────────────┘ │  │
│  │  ┌────────────────────────────────────────────────┐ │  │
│  │  │  Services:                                      │ │  │
│  │  │  - Auth Service (JWT generation/validation)    │ │  │
│  │  │  - LLM Service (OpenRouter integration)        │ │  │
│  │  │  - Conversation Session Manager                │ │  │
│  │  │  - Chat Handler (REST)                         │ │  │
│  │  └────────────────────────────────────────────────┘ │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                          │
                          │ HTTPS REST API
                          │ Authorization: Bearer <API_KEY>
                          ▼
┌─────────────────────────────────────────────────────────────┐
│               OpenRouter API (External)                      │
│  - Model: Configurable via OPENROUTER_MODEL env var        │
│  - Default: meta-llama/llama-3.3-8b-instruct:free          │
│  - System Prompt: Configurable via OPENROUTER_SYSTEM_PROMPT│
│  - Default: "You are a helpful assistant."                  │
│  - Endpoint: https://openrouter.ai/api/v1/chat/completions │
│  - Conversation: Full history sent with each request        │
└─────────────────────────────────────────────────────────────┘
```

### Key Features

1. **Theme Support**:
   - Light and Dark themes
   - Theme toggle button in the Chat interface (🌙/☀️)
   - Theme preference automatically saved to browser localStorage
   - Smooth transitions between themes
   - System preference detection (respects OS dark mode setting)

2. **Authentication Flow**:
   - User logs in with credentials (default: `demo`/`demo123`)
   - Backend generates JWT token valid for 24 hours
   - Token stored in localStorage
   - All protected endpoints require `Authorization: Bearer <token>` header

3. **Communication Patterns**:
   - **HTTP REST**: Standard REST API for non-streaming requests
   - **Server-Sent Events (SSE)**: Real-time streaming of LLM responses using SSE for responsive UX
   - **CORS**: Enabled for cross-origin requests
   - **JWT Authentication**: Bearer token authentication on protected endpoints

4. **Chat Flow**:
   - User types message and clicks Send
   - Frontend **immediately displays user message** (optimistic UI update)
   - Backend receives message via SSE streaming endpoint
   - Backend maintains server-side conversation history per user
   - Backend forwards full conversation history to OpenRouter API with `stream: true`
   - LLM starts streaming response chunks via SSE
   - Frontend receives chunks in real-time and builds response character-by-character
   - User sees the LLM response appearing progressively (streaming effect)
   - Once streaming completes, backend adds full response to conversation history
   - LLM has full context of previous messages in the conversation

5. **Security**:
   - JWT-based authentication
   - Token validation on protected routes
   - API key stored securely in environment variables

## How to Build

### Prerequisites

1. **Get OpenRouter API Key**:
   - Sign up at [OpenRouter](https://openrouter.ai/)
   - Get your API key from the dashboard
   - Copy `.env.example` to `.env`:
     ```bash
     cp .env.example .env
     ```
   - Edit `.env` and configure:
     ```
     OPENROUTER_API_KEY=your_actual_api_key_here
     OPENROUTER_MODEL=meta-llama/llama-3.3-8b-instruct:free                    # Optional, this is the default
     OPENROUTER_SYSTEM_PROMPT=You are a helpful assistant.                     # Optional, this is the default
     ```
   - You can use any model supported by OpenRouter (e.g., `anthropic/claude-3.5-sonnet`, `openai/gpt-4`, etc.)
   - You can customize the system prompt to change how the LLM behaves (e.g., "You are a Python expert" or "Answer in Spanish")

### Option 1: Build with Docker Compose (Recommended)

```bash
# Build both frontend and backend containers
docker compose build
```

### Option 2: Build Manually

#### Backend
```bash
cd backend

# Download dependencies
go mod download

# Build the binary
go build -o server ./cmd/server

# Or build for production (Linux)
CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server
```

#### Frontend
```bash
cd frontend

# Install dependencies
npm install

# Build for production
npm run build
```

## How to Run

### Option 1: Run with Docker Compose (Recommended)

```bash
# Make sure you have created .env file with OPENROUTER_API_KEY

# Start both services
docker compose up

# Or run in detached mode
docker compose up -d

# View logs
docker compose logs -f

# Stop services
docker compose down
```

Access the application:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Health check: http://localhost:8080/api/health

### Option 2: Run Manually

#### Terminal 1 - Backend
```bash
cd backend

# Set environment variables (Unix/Mac)
export OPENROUTER_API_KEY=your_api_key_here
export OPENROUTER_MODEL=meta-llama/llama-3.3-8b-instruct:free  # Optional
export OPENROUTER_SYSTEM_PROMPT="You are a helpful assistant."  # Optional

# Or for Windows
set OPENROUTER_API_KEY=your_api_key_here
set OPENROUTER_MODEL=meta-llama/llama-3.3-8b-instruct:free
set OPENROUTER_SYSTEM_PROMPT=You are a helpful assistant.

# Run the server
go run ./cmd/server/main.go

# Or run the built binary
./server
```

Backend will start on http://localhost:8080

#### Terminal 2 - Frontend
```bash
cd frontend

# Create .env file (if not exists)
echo "REACT_APP_API_URL=http://localhost:8080" > .env

# Start development server
npm start
```

Frontend will start on http://localhost:3000

## Conversation Context & Optimistic UI Updates

The application uses two key techniques for better UX and LLM context:

### Optimistic UI Updates
- User message appears **instantly** when sent (no waiting for server response)
- Provides immediate visual feedback to the user
- Frontend displays message immediately while backend processes it
- Users can continue typing without waiting for the LLM response
- Much more responsive than waiting for full API response

### Server-Side Conversation History
- Backend automatically maintains conversation history per user
- Each message is stored on the server (identified by username)
- Each new message is sent to OpenRouter with **full conversation context**
- LLM has complete context of the entire conversation

This enables:
- Follow-up questions that reference previous answers
- Multi-turn conversations with better context understanding
- More coherent and relevant responses from the LLM
- Responsive UI despite API latency (thanks to optimistic updates)
- Conversation persistence within a session

### Server-Sent Events (SSE) Streaming
The application uses **SSE (Server-Sent Events)** for streaming LLM responses instead of WebSockets because:
- **Simplicity**: SSE is built on standard HTTP, no additional protocol complexity
- **Compatibility**: Works through HTTP proxies and firewalls without special configuration
- **Unidirectional**: Perfect fit for server-to-client streaming (client sends message, server streams response)
- **Automatic Reconnection**: Browser handles reconnection logic automatically on disconnect
- **Standards-based**: W3C standard part of HTML5 Streams API
- **Efficiency**: Lower overhead than WebSockets for one-way communication
- **Progressive Response**: Users see LLM response appearing character-by-character as it's generated
- **Responsive UI**: Combined with optimistic message updates for instant feedback

## Usage

1. **Login**:
   - Open http://localhost:3000
   - Default credentials: `demo` / `demo123`
   - Click "Login"

2. **Chat**:
   - Type your message in the input field
   - Click "Send" or press Enter
   - Your message appears **instantly** in the chat (optimistic update)
   - AI response starts streaming immediately - you'll see it appearing character-by-character
   - The response builds progressively as the LLM generates tokens
   - Once complete, you can continue the conversation
   - The LLM has full context from previous messages in the conversation!

3. **Switch Theme**:
   - Click the moon (🌙) or sun (☀️) button in the header to toggle between light and dark themes
   - Your theme preference is automatically saved and will persist across sessions

4. **Logout**:
   - Click the "Logout" button in the top-right corner

## API Endpoints

### Public Endpoints
- `POST /api/login` - User authentication
  ```json
  Request: {"username": "demo", "password": "demo123"}
  Response: {"token": "eyJhbGc..."}
  ```
- `GET /api/health` - Health check

### Protected Endpoints (require JWT token)

- `POST /api/chat` - Send message with automatic conversation history (non-streaming)
  ```json
  Headers: {"Authorization": "Bearer <token>"}

  Request: {"message": "Hello"}
  Response: {
    "response": "Hi there! How can I help you?"
  }

  Notes:
  - Returns complete response in a single API call
  - Backend automatically maintains conversation history server-side (per user)
  - Backend sends full conversation history to OpenRouter with each request
  ```

- `POST /api/chat/stream` - Send message with streaming SSE response
  ```
  Headers: {"Authorization": "Bearer <token>"}
  Content-Type: text/event-stream

  Request: {"message": "Hello"}
  Response: Server-Sent Events stream of chunks
    data: Hi
    data:  there
    data: !
    data:  How
    data:  can
    data:  I
    data:  help
    data:  you
    data: ?
    data: [DONE]

  Notes:
  - Streams response chunks in real-time using SSE format
  - Browser automatically parses SSE events and updates UI progressively
  - Backend maintains full conversation history per user
  - Each chunk arrives as it's generated by the LLM
  - [DONE] marker signals end of stream
  - Perfect for responsive UI that shows progressive responses
  ```

## Theme System

The application includes a built-in light and dark theme system:

### Light Theme
- Light gray background with white surfaces
- Dark text for good readability
- Blue primary buttons and user message background

### Dark Theme
- Dark background with slightly lighter surfaces
- Light text for comfortable viewing in low light
- Blue primary buttons (adjusted for dark background)
- Dark message backgrounds with light text

### Implementation
- **Context API**: Theme state managed via React Context (`ThemeContext.tsx`)
- **Themes Configuration**: Centralized color definitions in `themes.ts`
- **Persistence**: Theme preference saved to browser localStorage
- **System Detection**: Automatically detects OS-level dark mode preference
- **Smooth Transitions**: CSS transitions for theme changes (0.3s ease)

## Project Structure

```
.
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # Entry point
│   ├── internal/
│   │   ├── auth/
│   │   │   └── auth.go              # JWT authentication
│   │   ├── conversation/
│   │   │   └── conversation.go      # Session & conversation history management
│   │   ├── handlers/
│   │   │   └── chat.go              # Chat handlers (REST + SSE streaming)
│   │   └── llm/
│   │       └── openrouter.go        # OpenRouter API integration (REST + SSE streaming)
│   ├── go.mod                       # Go dependencies
│   ├── Dockerfile                   # Backend container
│   └── .gitignore
├── frontend/
│   ├── public/
│   │   └── index.html
│   ├── src/
│   │   ├── components/
│   │   │   ├── Login.tsx            # Login component
│   │   │   └── Chat.tsx             # Chat component
│   │   ├── contexts/
│   │   │   └── ThemeContext.tsx      # Theme provider & hook
│   │   ├── services/
│   │   │   ├── auth.ts              # Auth service
│   │   │   └── chat.ts              # Chat service (REST + SSE streaming)
│   │   ├── App.tsx                  # Main app component
│   │   ├── index.tsx                # Entry point
│   │   ├── index.css                # Global styles
│   │   └── themes.ts                # Theme color definitions
│   ├── package.json                 # Node dependencies
│   ├── tsconfig.json                # TypeScript config
│   ├── nginx.conf                   # Nginx config for Docker
│   ├── Dockerfile                   # Frontend container
│   ├── .env.example                 # Example environment vars
│   └── .gitignore
├── docker-compose.yml               # Docker Compose config
├── .env.example                     # Example API key config
└── README.md                        # This file
```

## Troubleshooting

### Backend Issues
- **"OPENROUTER_API_KEY not configured"**: Make sure you set the environment variable
- **"Connection refused"**: Check if backend is running on port 8080
- **"Invalid token"**: Login again to get a new JWT token
- **Stream not connecting**: Make sure browser supports SSE (all modern browsers do). Check browser console for CORS errors
- **Incomplete streaming response**: The backend may have encountered an error. Check backend logs with `docker-compose logs backend`
- **Slow response time**: The LLM API response can take 1-5 seconds. This is normal and expected. Your message appears instantly due to optimistic UI updates, and response streams as it's generated
- **Model not working**: Check that `OPENROUTER_MODEL` is set to a valid model ID from OpenRouter. If not set, it defaults to `meta-llama/llama-3.3-8b-instruct:free`
- **LLM behavior not as expected**: Check the `OPENROUTER_SYSTEM_PROMPT` environment variable. Customize it to change how the LLM responds (e.g., "You are a helpful coding assistant" or "Respond in French")

### Frontend Issues
- **Can't connect to backend**: Update `REACT_APP_API_URL` in `.env`
- **CORS errors**: Make sure backend CORS is properly configured. SSE requires CORS headers
- **API calls fail**: Check that the backend is running and accessible on the configured URL
- **Stream stops before completion**: Check browser network tab - look for stream connection issues. Ensure network connection is stable
- **Response not appearing**: Check browser console for JavaScript errors. Clear browser cache and reload
- **Theme not persisting**: Check browser localStorage is enabled. If you clear browser data, theme preference will be reset
- **Theme toggle not working**: Make sure JavaScript is enabled in your browser
- **Conversation history empty**: Logout and login again to start a fresh conversation session

### Docker Issues
- **Port already in use**: Change ports in `docker-compose.yml`
- **Build fails**: Make sure you have enough disk space and Docker daemon is running
- **Container crashes**: Check logs with `docker-compose logs backend` or `docker-compose logs frontend`

## License

MIT
