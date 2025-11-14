package validation

import (
	"testing"
)

func TestAuthRequestValidator_ValidateUsername(t *testing.T) {
	validator := NewAuthRequestValidator()

	tests := []struct {
		name     string
		username string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid username",
			username: "testuser",
			wantErr:  false,
		},
		{
			name:     "valid username with numbers",
			username: "user123",
			wantErr:  false,
		},
		{
			name:     "valid username with underscore",
			username: "test_user",
			wantErr:  false,
		},
		{
			name:     "valid username with hyphen",
			username: "test-user",
			wantErr:  false,
		},
		{
			name:     "minimum length username",
			username: "abc",
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			wantErr:  true,
			errMsg:   "username cannot be empty",
		},
		{
			name:     "username too short",
			username: "ab",
			wantErr:  true,
			errMsg:   "username must be at least 3 characters long",
		},
		{
			name:     "username too long",
			username: "a123456789012345678901234567890123456789012345678901",
			wantErr:  true,
			errMsg:   "username must be at most 50 characters long",
		},
		{
			name:     "username with spaces",
			username: "test user",
			wantErr:  true,
			errMsg:   "username can only contain letters, numbers, underscores, and hyphens",
		},
		{
			name:     "username with special characters",
			username: "test@user",
			wantErr:  true,
			errMsg:   "username can only contain letters, numbers, underscores, and hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateUsername() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestAuthRequestValidator_ValidatePassword(t *testing.T) {
	validator := NewAuthRequestValidator()

	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "minimum length password",
			password: "123456",
			wantErr:  false,
		},
		{
			name:     "password with special characters",
			password: "P@ssw0rd!",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			errMsg:   "password cannot be empty",
		},
		{
			name:     "password too short",
			password: "12345",
			wantErr:  true,
			errMsg:   "password must be at least 6 characters long",
		},
		{
			name:     "password too long",
			password: string(make([]byte, 129)),
			wantErr:  true,
			errMsg:   "password must be at most 128 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePassword() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestAuthRequestValidator_ValidateEmail(t *testing.T) {
	validator := NewAuthRequestValidator()

	tests := []struct {
		name    string
		email   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "valid email with plus",
			email:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with hyphen",
			email:   "user@ex-ample.com",
			wantErr: false,
		},
		{
			name:    "empty email (optional)",
			email:   "",
			wantErr: false,
		},
		{
			name:    "invalid email - no @",
			email:   "userexample.com",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - no domain",
			email:   "user@",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - no local part",
			email:   "@example.com",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - no TLD",
			email:   "user@example",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "email too long",
			email:   "verylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddressverylongemailaddress@example.com",
			wantErr: true,
			errMsg:  "email must be at most 255 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEmail() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestAuthRequestValidator_ValidateLoginRequest(t *testing.T) {
	validator := NewAuthRequestValidator()

	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid login request",
			username: "testuser",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			password: "password123",
			wantErr:  true,
			errMsg:   "username cannot be empty",
		},
		{
			name:     "empty password",
			username: "testuser",
			password: "",
			wantErr:  true,
			errMsg:   "password cannot be empty",
		},
		{
			name:     "both empty",
			username: "",
			password: "",
			wantErr:  true,
			errMsg:   "username cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLoginRequest(tt.username, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLoginRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateLoginRequest() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestAuthRequestValidator_ValidateRegisterRequest(t *testing.T) {
	validator := NewAuthRequestValidator()

	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid registration request",
			username: "testuser",
			email:    "test@example.com",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "valid registration without email",
			username: "testuser",
			email:    "",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "invalid username",
			username: "ab",
			email:    "test@example.com",
			password: "password123",
			wantErr:  true,
			errMsg:   "username must be at least 3 characters long",
		},
		{
			name:     "invalid email",
			username: "testuser",
			email:    "invalid-email",
			password: "password123",
			wantErr:  true,
			errMsg:   "invalid email format",
		},
		{
			name:     "invalid password",
			username: "testuser",
			email:    "test@example.com",
			password: "12345",
			wantErr:  true,
			errMsg:   "password must be at least 6 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRegisterRequest(tt.username, tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegisterRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateRegisterRequest() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
