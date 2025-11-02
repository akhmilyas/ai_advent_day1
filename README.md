# AI Chat Application

A fullstack chat application with Go backend and React TypeScript frontend, featuring real-time streaming responses from OpenRouter's LLM APIs.

## Dependencies

### Backend
- **Go**: 1.21 or higher
- **Dependencies** (managed via go.mod):
  - `github.com/coder/websocket` v1.8.12 - WebSocket implementation
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
│  │  - Chat Service (HTTP REST & WebSocket)             │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                          │
                          │ HTTP REST (auth, non-streaming)
                          │ WebSocket (streaming chat)
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
│  │  │  WS   /api/chat/stream (protected, WebSocket)  │ │  │
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
│  │  │  - Chat Handlers (REST & WebSocket)            │ │  │
│  │  └────────────────────────────────────────────────┘ │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                          │
                          │ HTTPS REST API
                          │ Authorization: Bearer <API_KEY>
                          ▼
┌─────────────────────────────────────────────────────────────┐
│               OpenRouter API (External)                      │
│  - Model: meta-llama/llama-3.1-8b-instruct:free            │
│  - Endpoint: https://openrouter.ai/api/v1/chat/completions │
│  - Streaming: Server-Sent Events (SSE)                      │
└─────────────────────────────────────────────────────────────┘
```

### Key Features

1. **Authentication Flow**:
   - User logs in with credentials (default: `demo`/`demo123`)
   - Backend generates JWT token valid for 24 hours
   - Token stored in localStorage
   - All protected endpoints require `Authorization: Bearer <token>` header

2. **Communication Patterns**:
   - **HTTP REST**: Used for login and non-streaming chat
   - **WebSocket**: Used for real-time streaming responses from LLM
   - **CORS**: Enabled for cross-origin requests

3. **Chat Flow**:
   - User sends message via WebSocket
   - Backend forwards to OpenRouter API with streaming enabled
   - Response chunks streamed back to frontend via WebSocket
   - Frontend displays chunks in real-time with typing animation

4. **Security**:
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
   - Edit `.env` and add your API key:
     ```
     OPENROUTER_API_KEY=your_actual_api_key_here
     ```

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

# Set environment variable (Unix/Mac)
export OPENROUTER_API_KEY=your_api_key_here

# Or for Windows
set OPENROUTER_API_KEY=your_api_key_here

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
echo "REACT_APP_WS_URL=ws://localhost:8080" >> .env

# Start development server
npm start
```

Frontend will start on http://localhost:3000

## Usage

1. **Login**:
   - Open http://localhost:3000
   - Default credentials: `demo` / `demo123`
   - Click "Login"

2. **Chat**:
   - Type your message in the input field
   - Click "Send" or press Enter
   - Watch the AI response stream in real-time
   - The streaming indicator (blinking cursor) shows when AI is responding

3. **Logout**:
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
- `POST /api/chat` - Send message (non-streaming)
  ```json
  Headers: {"Authorization": "Bearer <token>"}
  Request: {"message": "Hello"}
  Response: {"response": "Hi there!"}
  ```
- `WS /api/chat/stream` - WebSocket for streaming chat
  ```json
  Send: {"message": "Hello"}
  Receive: {"type": "start", "content": ""}
  Receive: {"type": "chunk", "content": "Hi"}
  Receive: {"type": "chunk", "content": " there"}
  Receive: {"type": "end", "content": ""}
  ```

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
│   │   ├── handlers/
│   │   │   └── chat.go              # HTTP & WebSocket handlers
│   │   └── llm/
│   │       └── openrouter.go        # OpenRouter API integration
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
│   │   ├── services/
│   │   │   ├── auth.ts              # Auth service
│   │   │   └── chat.ts              # Chat service with WS
│   │   ├── App.tsx                  # Main app component
│   │   ├── index.tsx                # Entry point
│   │   └── index.css                # Global styles
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

### Frontend Issues
- **Can't connect to backend**: Update `REACT_APP_API_URL` and `REACT_APP_WS_URL` in `.env`
- **CORS errors**: Make sure backend CORS is properly configured
- **WebSocket connection fails**: Check firewall settings and ensure WebSocket upgrade is allowed

### Docker Issues
- **Port already in use**: Change ports in `docker-compose.yml`
- **Build fails**: Make sure you have enough disk space and Docker daemon is running
- **Container crashes**: Check logs with `docker-compose logs backend` or `docker-compose logs frontend`

## License

MIT
