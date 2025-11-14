package auth

import (
	"chat-app/internal/config"
	"chat-app/internal/db"
	"chat-app/internal/logger"
	"chat-app/internal/validation"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthHandlers holds handlers with configuration
type AuthHandlers struct {
	config    *config.AppConfig
	db        db.Database
	validator *validation.AuthRequestValidator
}

// NewAuthHandlers creates auth handlers with config
func NewAuthHandlers(appConfig *config.AppConfig, database db.Database) *AuthHandlers {
	return &AuthHandlers{
		config:    appConfig,
		db:        database,
		validator: validation.NewAuthRequestValidator(),
	}
}

// getJWTSecret retrieves the JWT secret from config
func (h *AuthHandlers) getJWTSecret() []byte {
	return h.config.Auth.JWTSecret
}

// getTokenExpiration retrieves token expiration from config
func (h *AuthHandlers) getTokenExpiration() time.Duration {
	return h.config.Auth.TokenExpiration
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// sendError sends a standardized JSON error response
func sendError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errResp := ErrorResponse{
		Code:    status,
		Message: message,
	}
	if err != nil {
		errResp.Error = err.Error()
	}
	json.NewEncoder(w).Encode(errResp)
}

func (h *AuthHandlers) GenerateToken(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.getTokenExpiration())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.getJWTSecret())
}

func (h *AuthHandlers) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return h.getJWTSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// LoginHandler authenticates user and returns JWT token
func (h *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validator.ValidateLoginRequest(req.Username, req.Password); err != nil {
		sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Get user from database
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"username": req.Username,
			"error":    err.Error(),
		}).Warn("Login failed: user not found")
		sendError(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Verify password
	if !user.VerifyPassword(req.Password) {
		logger.Log.WithFields(logrus.Fields{
			"username": req.Username,
		}).Warn("Login failed: invalid password")
		sendError(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Generate JWT token
	token, err := h.GenerateToken(req.Username)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"username": req.Username,
			"error":    err.Error(),
		}).Error("Error generating token")
		sendError(w, http.StatusInternalServerError, "Error generating token", err)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": req.Username,
	}).Info("User logged in successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

// RegisterHandler creates a new user account
func (h *AuthHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validator.ValidateRegisterRequest(req.Username, req.Email, req.Password); err != nil {
		sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Create user in database
	user, err := h.db.CreateUser(req.Username, req.Email, req.Password)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"username": req.Username,
			"error":    err.Error(),
		}).Warn("Registration failed")
		if err.Error() == "username already exists" {
			sendError(w, http.StatusConflict, "Username already exists", err)
			return
		}
		sendError(w, http.StatusInternalServerError, "Error creating user", err)
		return
	}

	// Generate JWT token
	token, err := h.GenerateToken(user.Username)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"username": user.Username,
			"error":    err.Error(),
		}).Error("Error generating token")
		sendError(w, http.StatusInternalServerError, "Error generating token", err)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"email":    user.Email,
	}).Info("User registered successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(RegisterResponse{
		Message: "User registered successfully",
		Token:   token,
	})
}

func (h *AuthHandlers) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendError(w, http.StatusUnauthorized, "Missing authorization header", nil)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			sendError(w, http.StatusUnauthorized, "Invalid authorization header format", nil)
			return
		}

		claims, err := h.ValidateToken(bearerToken[1])
		if err != nil {
			sendError(w, http.StatusUnauthorized, "Invalid token", err)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
