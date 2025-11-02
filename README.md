# AI Chat Application

A fullstack chat application with Go backend and React TypeScript frontend, featuring:
- User registration and authentication with JWT tokens
- REST API integration with OpenRouter's LLM APIs
- Server-side conversation history persistence in PostgreSQL
- Real-time response streaming with Server-Sent Events (SSE)
- Dark/Light theme support

## Dependencies

### Backend
- **Go**: 1.21 or higher
- **PostgreSQL**: 13 or higher (automatically managed by Docker Compose)
- **Dependencies** (managed via go.mod):
  - `github.com/golang-jwt/jwt/v5` v5.2.0 - JWT authentication
  - `github.com/lib/pq` v1.10.9 - PostgreSQL driver
  - `golang.org/x/crypto` v0.17.0 - Password hashing (bcrypt)

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
│  │  - Login/Register Components (JWT Authentication)   │  │
│  │  - Chat Component (Message UI)                       │  │
│  │  - Auth Service (Token Management)                   │  │
│  │  - Chat Service (HTTP REST + SSE Streaming)          │  │
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
│  │  │  POST /api/register (public)                   │ │  │
│  │  │  GET  /api/health (public)                     │ │  │
│  │  │  POST /api/chat (protected, REST)              │ │  │
│  │  │  POST /api/chat/stream (protected, SSE)        │ │  │
│  │  │  GET  /api/conversations (protected)           │ │  │
│  │  └────────────────────────────────────────────────┘ │  │
│  │  ┌────────────────────────────────────────────────┐ │  │
│  │  │  Middleware:                                    │ │  │
│  │  │  - CORS Handler                                 │ │  │
│  │  │  - JWT Authentication                           │ │  │
│  │  └────────────────────────────────────────────────┘ │  │
│  │  ┌────────────────────────────────────────────────┐ │  │
│  │  │  Services:                                      │ │  │
│  │  │  - Auth Service (JWT & user authentication)    │ │  │
│  │  │  - LLM Service (OpenRouter integration)        │ │  │
│  │  │  - Database Service (user & conversation mgmt) │ │  │
│  │  │  - Chat Handler (REST + SSE streaming)         │ │  │
│  │  └────────────────────────────────────────────────┘ │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
              │                              │
              │                              │ SQL Queries
              │ HTTPS REST API               │
              │ Authorization: Bearer        ▼
              │ <API_KEY>          ┌──────────────────────┐
              │                    │   PostgreSQL (DB)    │
              │                    │   Port 5432          │
              │                    │  - users table       │
              │                    │  - conversations tbl │
              │                    │  - messages table    │
              │                    └──────────────────────┘
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

1. **User Management & Registration**:
   - Intuitive registration form with toggle to login
   - User registration with username, email (optional), and password
   - Password validation: minimum 6 characters, must be confirmed
   - Passwords securely hashed with bcrypt before storage
   - Pre-seeded demo user account for quick testing (demo/demo123)
   - All users stored in PostgreSQL database with unique usernames
   - User sessions last 24 hours with JWT bearer tokens
   - Seamless toggle between login and registration modes

2. **Persistent Conversation History**:
   - All conversations and messages stored in PostgreSQL
   - Each conversation automatically titled with first message
   - Full conversation history persists across sessions
   - Users can access all their previous conversations
   - Messages ordered chronologically within conversations

3. **Theme Support**:
   - Light and Dark themes
   - Theme toggle button in the Chat interface (🌙/☀️)
   - Theme preference automatically saved to browser localStorage
   - Smooth transitions between themes
   - System preference detection (respects OS dark mode setting)

4. **Authentication Flow**:
   - Users can register new accounts or login with existing credentials
   - Backend generates JWT token valid for 24 hours
   - Token stored in localStorage
   - All protected endpoints require `Authorization: Bearer <token>` header
   - Passwords never sent to frontend, always hashed server-side

5. **Communication Patterns**:
   - **HTTP REST**: Standard REST API for non-streaming requests
   - **Server-Sent Events (SSE)**: Real-time streaming of LLM responses using SSE for responsive UX
   - **CORS**: Enabled for cross-origin requests
   - **JWT Authentication**: Bearer token authentication on protected endpoints

6. **Chat Flow**:
   - User types message and clicks Send
   - Frontend **immediately displays user message** (optimistic UI update)
   - Backend creates new conversation if needed (first message titles it)
   - Message stored in database
   - Backend forwards full conversation history to OpenRouter API with `stream: true`
   - LLM starts streaming response chunks via SSE
   - Frontend receives chunks in real-time and builds response character-by-character
   - User sees the LLM response appearing progressively (streaming effect)
   - Once streaming completes, backend adds full response to database
   - Next message continues conversation with full context

7. **Security**:
   - Password hashing with bcrypt (industry standard)
   - JWT-based authentication with configurable expiry
   - Token validation on protected routes
   - API key stored securely in environment variables
   - Database uses authenticated connections only

## How to Build

### Prerequisites

1. **Get OpenRouter API Key**:
   - Sign up at [OpenRouter](https://openrouter.ai/)
   - Get your API key from the dashboard

2. **Configure Environment**:
   - Copy `.env.example` to `.env`:
     ```bash
     cp .env.example .env
     ```
   - Edit `.env` and configure (required and optional):
     ```bash
     # Required
     OPENROUTER_API_KEY=your_actual_api_key_here

     # Optional LLM Configuration
     OPENROUTER_MODEL=meta-llama/llama-3.3-8b-instruct:free  # Defaults to this if not set
     OPENROUTER_SYSTEM_PROMPT=You are a helpful assistant.   # Defaults to this if not set

     # Optional Database Configuration
     # When using Docker Compose, these default to the values shown below
     DB_HOST=postgres           # Use 'postgres' for Docker, 'localhost' for local dev
     DB_PORT=5432
     DB_USER=postgres
     DB_PASSWORD=postgres
     DB_NAME=chatapp
     DB_SSLMODE=disable
     ```
   - You can customize:
     - **Model**: Use any model from OpenRouter (e.g., `anthropic/claude-3.5-sonnet`, `openai/gpt-4`, etc.)
     - **System Prompt**: Change LLM behavior (e.g., "You are a Python expert" or "Respond in Spanish")
     - **Database**: For local development without Docker, change `DB_HOST` to `localhost`

### Option 1: Build with Docker Compose (Recommended)

```bash
# Build both frontend and backend containers
docker compose build
```

### Option 2: Build Manually

#### Prerequisites for Manual Build
- **PostgreSQL 13+** must be installed and running locally
- Create database and user:
  ```bash
  # Using psql command line
  createdb chatapp
  createuser postgres
  psql -U postgres -d chatapp
  # Database tables will be created automatically on first backend run
  ```

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
export OPENROUTER_MODEL=meta-llama/llama-3.3-8b-instruct:free        # Optional
export OPENROUTER_SYSTEM_PROMPT="You are a helpful assistant."       # Optional

# Database configuration (for local PostgreSQL)
export DB_HOST=localhost       # Default: postgres (Docker)
export DB_PORT=5432          # Default: 5432
export DB_USER=postgres       # Default: postgres
export DB_PASSWORD=postgres   # Default: postgres
export DB_NAME=chatapp        # Default: chatapp
export DB_SSLMODE=disable     # Default: disable

# Or for Windows
set OPENROUTER_API_KEY=your_api_key_here
set OPENROUTER_MODEL=meta-llama/llama-3.3-8b-instruct:free
set OPENROUTER_SYSTEM_PROMPT=You are a helpful assistant.
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=postgres
set DB_NAME=chatapp
set DB_SSLMODE=disable

# Run the server (automatically creates tables on first run)
go run ./cmd/server/main.go

# Or run the built binary
./server
```

Backend will start on http://localhost:8080. On first run, it will:
- Connect to PostgreSQL
- Create `users`, `conversations`, and `messages` tables
- Seed the demo user account (username: demo, password: demo123)

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

1. **Authentication**:
   - **Option A - Login**: Open http://localhost:3000 and use default credentials: `demo` / `demo123`
   - **Option B - Register**: Click "Register here" to create a new account
     - Enter a unique username
     - Enter your email (optional)
     - Enter a password (minimum 6 characters)
     - Confirm your password
     - Click "Register"
     - You'll be immediately logged in with your new account
   - Both login and registration forms are accessible via toggle links

2. **Chat**:
   - Type your message in the input field at the bottom
   - Click "Send" or press Enter to send
   - Your message appears **instantly** in the chat (optimistic update - no waiting)
   - AI response starts streaming immediately - you'll see it appearing character-by-character
   - The response builds progressively as the LLM generates tokens
   - Once complete, you can continue the conversation
   - The LLM has full context from all previous messages in the conversation!
   - Each conversation is automatically titled with your first message and persists across sessions

3. **Conversations**:
   - Each new conversation is automatically created and titled
   - Your conversation history is preserved in the database
   - Switch between conversations by logging out and back in (future enhancement: conversation switcher)
   - All messages are stored and restored when you return

4. **Switch Theme**:
   - Click the moon (🌙) or sun (☀️) button in the header to toggle between light and dark themes
   - Your theme preference is automatically saved and will persist across sessions

5. **Logout**:
   - Click the "Logout" button in the top-right corner
   - Your conversations and account are preserved and available when you log back in

## API Endpoints

### Public Endpoints

- `POST /api/login` - User authentication
  ```json
  Request: {"username": "demo", "password": "demo123"}
  Response: {"token": "eyJhbGc..."}
  ```
  Status codes: 200 (success), 400 (missing fields), 401 (invalid credentials)

- `POST /api/register` - Create new user account
  ```json
  Request: {
    "username": "newuser",
    "email": "user@example.com",
    "password": "securepassword123"
  }
  Response: {
    "message": "User registered successfully",
    "token": "eyJhbGc..."
  }
  ```
  Status codes: 201 (created), 400 (validation error), 409 (username exists)

  Frontend Validation:
  - Username: Required, must be unique
  - Email: Optional
  - Password: Required, minimum 6 characters
  - Password Confirmation: Must match password field

  Backend Validation:
  - Username uniqueness enforced at database level
  - Password hashed with bcrypt before storage
  - Returns JWT token for immediate login after successful registration
  - New users can access chat immediately with their new account

- `GET /api/health` - Health check
  ```
  Response: OK
  Status: 200
  ```

### Protected Endpoints (require JWT token in Authorization header)

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
  - Optional `conversation_id` in request body to continue existing conversation
  ```

- `GET /api/conversations` - Get all conversations for authenticated user
  ```json
  Headers: {"Authorization": "Bearer <token>"}

  Response: {
    "conversations": [
      {
        "id": 1,
        "title": "First message of conversation",
        "created_at": "2024-11-02T10:30:00Z",
        "updated_at": "2024-11-02T10:35:00Z"
      }
    ]
  }
  ```
  Notes:
  - Returns list of all user's conversations ordered by most recent first
  - Title automatically set to first message (max 100 chars)
  - Use `conversation_id` in chat requests to continue existing conversations
  - Status: 200 (success), 401 (unauthorized), 500 (error)
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
│   │       └── main.go              # Entry point (initializes DB, seeds demo user)
│   ├── internal/
│   │   ├── auth/
│   │   │   └── auth.go              # JWT & user authentication (login/register)
│   │   ├── db/
│   │   │   ├── postgres.go          # PostgreSQL connection & schema initialization
│   │   │   ├── user.go              # User repository with password hashing
│   │   │   └── conversation.go      # Conversation & message persistence
│   │   ├── handlers/
│   │   │   └── chat.go              # Chat handlers (REST + SSE, now with DB)
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
│   │   │   ├── Login.tsx            # Login & Registration component (toggleable forms)
│   │   │   └── Chat.tsx             # Chat component with SSE streaming
│   │   ├── contexts/
│   │   │   └── ThemeContext.tsx      # Theme provider & hook
│   │   ├── services/
│   │   │   ├── auth.ts              # Auth service (login & register methods)
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

#### LLM & API Issues
- **"OPENROUTER_API_KEY not configured"**: Make sure you set the environment variable
- **Model not working**: Check that `OPENROUTER_MODEL` is set to a valid model ID from OpenRouter. If not set, it defaults to `meta-llama/llama-3.3-8b-instruct:free`
- **LLM behavior not as expected**: Check the `OPENROUTER_SYSTEM_PROMPT` environment variable. Customize it to change how the LLM responds (e.g., "You are a helpful coding assistant" or "Respond in French")

#### Streaming & Connection Issues
- **Stream not connecting**: Make sure browser supports SSE (all modern browsers do). Check browser console for CORS errors
- **Incomplete streaming response**: The backend may have encountered an error. Check backend logs with `docker-compose logs backend`
- **Slow response time**: The LLM API response can take 1-5 seconds. This is normal and expected. Your message appears instantly due to optimistic UI updates, and response streams as it's generated
- **"Connection refused"**: Check if backend is running on port 8080

#### Database Issues
- **"Failed to initialize database" on startup**:
  - Make sure PostgreSQL is running (check with `docker-compose ps`)
  - For Docker Compose: PostgreSQL container should start automatically
  - For local dev: Ensure PostgreSQL is installed and running on localhost:5432
  - Check database credentials in `.env` match PostgreSQL setup
- **"error connecting to database"**: Check `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` environment variables
- **Tables already exist**: This is normal. The application safely creates tables only if they don't exist
- **"username already exists"** when registering: Try a different username. Use demo/demo123 if testing login
- **"User not found"** during login: Check username spelling. Demo user is pre-seeded at startup
- **Lost conversation history**: Conversations persist in the database. If you clear the database, history is lost. For Docker, use `docker-compose down --volumes` to reset

#### Authentication Issues
- **"Invalid token"**: Login again to get a new JWT token. Tokens expire after 24 hours
- **Login fails**: Check username and password are correct. Default demo user is demo/demo123
- **Registration errors**:
  - **"Username already exists"**: Username is taken. Choose a different username
  - **"Password must be at least 6 characters"**: Password is too short. Use at least 6 characters
  - **"Passwords do not match"**: The two password fields don't match. Re-enter and confirm password
  - **"Username is required"**: Enter a username in the username field
  - **"Email is required"**: Email field is mandatory (or leave it blank if optional is desired)
- **Can't toggle between login and register**: Click the blue "Register here" or "Login here" link text to switch modes
- **Lost access to account**: Passwords can't be reset yet. You'll need to create a new account or contact admin

### Frontend Issues
- **Can't connect to backend**: Update `REACT_APP_API_URL` in `.env`
- **CORS errors**: Make sure backend CORS is properly configured. SSE requires CORS headers
- **API calls fail**: Check that the backend is running and accessible on the configured URL
- **Stream stops before completion**: Check browser network tab - look for stream connection issues. Ensure network connection is stable
- **Response not appearing**: Check browser console for JavaScript errors. Clear browser cache and reload
- **Theme not persisting**: Check browser localStorage is enabled. If you clear browser data, theme preference will be reset
- **Theme toggle not working**: Make sure JavaScript is enabled in your browser
- **Conversation history empty after login**: This is expected on first login. Conversations are created when you send your first message and persist in the database
- **Can't see previous conversations**: Click "Get Conversations" to load your conversation list. All messages persist in the database and load when you return to a conversation

### Docker Issues
- **Port already in use**: Change ports in `docker-compose.yml`
- **Build fails**: Make sure you have enough disk space and Docker daemon is running
- **Container crashes**: Check logs with `docker-compose logs backend` or `docker-compose logs frontend`

## License

MIT
