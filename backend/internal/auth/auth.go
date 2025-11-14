package auth

import (
	"chat-app/internal/db"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"

var jwtSecret []byte

// getJWTSecret retrieves and validates the JWT secret from environment variable
func getJWTSecret() []byte {
	if jwtSecret != nil {
		return jwtSecret
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable must be set")
	}
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters")
	}
	jwtSecret = []byte(secret)
	return jwtSecret
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

func GenerateToken(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
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
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Username == "" || req.Password == "" {
		sendError(w, http.StatusBadRequest, "Username and password are required", nil)
		return
	}

	// Get user from database
	user, err := db.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("[AUTH] Login failed for user %s: user not found", req.Username)
		sendError(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Verify password
	if !user.VerifyPassword(req.Password) {
		log.Printf("[AUTH] Login failed for user %s: invalid password", req.Username)
		sendError(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Generate JWT token
	token, err := GenerateToken(req.Username)
	if err != nil {
		log.Printf("[AUTH] Error generating token: %v", err)
		sendError(w, http.StatusInternalServerError, "Error generating token", err)
		return
	}

	log.Printf("[AUTH] User %s logged in successfully", req.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

// RegisterHandler creates a new user account
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Username == "" || req.Password == "" {
		sendError(w, http.StatusBadRequest, "Username and password are required", nil)
		return
	}

	if len(req.Password) < 6 {
		sendError(w, http.StatusBadRequest, "Password must be at least 6 characters", nil)
		return
	}

	// Create user in database
	user, err := db.CreateUser(req.Username, req.Email, req.Password)
	if err != nil {
		log.Printf("[AUTH] Registration failed for user %s: %v", req.Username, err)
		if err.Error() == "username already exists" {
			sendError(w, http.StatusConflict, "Username already exists", err)
			return
		}
		sendError(w, http.StatusInternalServerError, "Error creating user", err)
		return
	}

	// Generate JWT token
	token, err := GenerateToken(user.Username)
	if err != nil {
		log.Printf("[AUTH] Error generating token: %v", err)
		sendError(w, http.StatusInternalServerError, "Error generating token", err)
		return
	}

	log.Printf("[AUTH] User %s registered successfully", user.Username)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(RegisterResponse{
		Message: "User registered successfully",
		Token:   token,
	})
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
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

		claims, err := ValidateToken(bearerToken[1])
		if err != nil {
			sendError(w, http.StatusUnauthorized, "Invalid token", err)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
