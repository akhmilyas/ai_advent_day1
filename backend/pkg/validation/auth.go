package validation

import (
	"errors"
	"fmt"
	"regexp"
)

// AuthRequestValidator validates authentication-related requests
type AuthRequestValidator struct{}

// NewAuthRequestValidator creates a new AuthRequestValidator
func NewAuthRequestValidator() *AuthRequestValidator {
	return &AuthRequestValidator{}
}

// ValidateUsername validates a username
func (v *AuthRequestValidator) ValidateUsername(username string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}

	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long, got %d", len(username))
	}

	if len(username) > 50 {
		return fmt.Errorf("username must be at most 50 characters long, got %d", len(username))
	}

	// Username should contain only alphanumeric characters, underscores, and hyphens
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUsername.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores, and hyphens")
	}

	return nil
}

// ValidatePassword validates a password
func (v *AuthRequestValidator) ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}

	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters long, got %d", len(password))
	}

	if len(password) > 128 {
		return fmt.Errorf("password must be at most 128 characters long, got %d", len(password))
	}

	return nil
}

// ValidateEmail validates an email address (basic validation)
func (v *AuthRequestValidator) ValidateEmail(email string) error {
	// Email is optional for registration
	if email == "" {
		return nil
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	if len(email) > 255 {
		return fmt.Errorf("email must be at most 255 characters long, got %d", len(email))
	}

	return nil
}

// ValidateLoginRequest validates a login request
func (v *AuthRequestValidator) ValidateLoginRequest(username, password string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}

	if password == "" {
		return errors.New("password cannot be empty")
	}

	return nil
}

// ValidateRegisterRequest validates a registration request
func (v *AuthRequestValidator) ValidateRegisterRequest(username, email, password string) error {
	if err := v.ValidateUsername(username); err != nil {
		return err
	}

	if err := v.ValidateEmail(email); err != nil {
		return err
	}

	if err := v.ValidatePassword(password); err != nil {
		return err
	}

	return nil
}
