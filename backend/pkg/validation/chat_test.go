package validation

import (
	"testing"
)

func TestChatRequestValidator_ValidateMessage(t *testing.T) {
	validator := NewChatRequestValidator()

	tests := []struct {
		name    string
		message string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid message",
			message: "Hello, world!",
			wantErr: false,
		},
		{
			name:    "valid long message",
			message: "This is a very long message with lots of content to test the validation",
			wantErr: false,
		},
		{
			name:    "valid message with special characters",
			message: "Test!@#$%^&*()",
			wantErr: false,
		},
		{
			name:    "empty message",
			message: "",
			wantErr: true,
			errMsg:  "message cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateMessage() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestChatRequestValidator_ValidateWarAndPeacePercent(t *testing.T) {
	validator := NewChatRequestValidator()

	tests := []struct {
		name    string
		percent int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid percent - 0",
			percent: 0,
			wantErr: false,
		},
		{
			name:    "valid percent - 50",
			percent: 50,
			wantErr: false,
		},
		{
			name:    "valid percent - 100",
			percent: 100,
			wantErr: false,
		},
		{
			name:    "invalid percent - negative",
			percent: -1,
			wantErr: true,
			errMsg:  "war_and_peace_percent must be between 0 and 100",
		},
		{
			name:    "invalid percent - too high",
			percent: 101,
			wantErr: true,
			errMsg:  "war_and_peace_percent must be between 0 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWarAndPeacePercent(tt.percent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWarAndPeacePercent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateWarAndPeacePercent() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestChatRequestValidator_ValidateTemperature(t *testing.T) {
	validator := NewChatRequestValidator()

	temp0 := 0.0
	temp07 := 0.7
	temp2 := 2.0
	tempNegative := -0.1
	tempTooHigh := 2.1

	tests := []struct {
		name        string
		temperature *float64
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "nil temperature (optional)",
			temperature: nil,
			wantErr:     false,
		},
		{
			name:        "valid temperature - 0",
			temperature: &temp0,
			wantErr:     false,
		},
		{
			name:        "valid temperature - 0.7",
			temperature: &temp07,
			wantErr:     false,
		},
		{
			name:        "valid temperature - 2",
			temperature: &temp2,
			wantErr:     false,
		},
		{
			name:        "invalid temperature - negative",
			temperature: &tempNegative,
			wantErr:     true,
			errMsg:      "temperature must be between 0 and 2",
		},
		{
			name:        "invalid temperature - too high",
			temperature: &tempTooHigh,
			wantErr:     true,
			errMsg:      "temperature must be between 0 and 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTemperature(tt.temperature)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemperature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateTemperature() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestChatRequestValidator_ValidateResponseFormat(t *testing.T) {
	validator := NewChatRequestValidator()

	tests := []struct {
		name    string
		format  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid format - text",
			format:  "text",
			wantErr: false,
		},
		{
			name:    "valid format - json",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "valid format - xml",
			format:  "xml",
			wantErr: false,
		},
		{
			name:    "empty format (optional, defaults to text)",
			format:  "",
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  "yaml",
			wantErr: true,
			errMsg:  "response_format must be one of: text, json, xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateResponseFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResponseFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateResponseFormat() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestChatRequestValidator_ValidateResponseSchema(t *testing.T) {
	validator := NewChatRequestValidator()

	tests := []struct {
		name    string
		format  string
		schema  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "text format - no schema required",
			format:  "text",
			schema:  "",
			wantErr: false,
		},
		{
			name:    "empty format - no schema required",
			format:  "",
			schema:  "",
			wantErr: false,
		},
		{
			name:    "json format with schema",
			format:  "json",
			schema:  `{"type": "object"}`,
			wantErr: false,
		},
		{
			name:    "xml format with schema",
			format:  "xml",
			schema:  `<root></root>`,
			wantErr: false,
		},
		{
			name:    "json format without schema",
			format:  "json",
			schema:  "",
			wantErr: true,
			errMsg:  "response_schema is required for json format",
		},
		{
			name:    "xml format without schema",
			format:  "xml",
			schema:  "",
			wantErr: true,
			errMsg:  "response_schema is required for xml format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateResponseSchema(tt.format, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResponseSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateResponseSchema() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestChatRequestValidator_ValidateChatRequest(t *testing.T) {
	validator := NewChatRequestValidator()

	temp07 := 0.7
	tempInvalid := 3.0

	tests := []struct {
		name               string
		message            string
		temperature        *float64
		warAndPeacePercent int
		responseFormat     string
		responseSchema     string
		wantErr            bool
		errMsg             string
	}{
		{
			name:               "valid chat request - text format",
			message:            "Hello",
			temperature:        &temp07,
			warAndPeacePercent: 50,
			responseFormat:     "text",
			responseSchema:     "",
			wantErr:            false,
		},
		{
			name:               "valid chat request - json format with schema",
			message:            "Hello",
			temperature:        &temp07,
			warAndPeacePercent: 50,
			responseFormat:     "json",
			responseSchema:     `{"type": "object"}`,
			wantErr:            false,
		},
		{
			name:               "valid chat request - xml format with schema",
			message:            "Hello",
			temperature:        &temp07,
			warAndPeacePercent: 50,
			responseFormat:     "xml",
			responseSchema:     `<root></root>`,
			wantErr:            false,
		},
		{
			name:               "invalid message",
			message:            "",
			temperature:        &temp07,
			warAndPeacePercent: 50,
			responseFormat:     "text",
			responseSchema:     "",
			wantErr:            true,
			errMsg:             "message cannot be empty",
		},
		{
			name:               "invalid temperature",
			message:            "Hello",
			temperature:        &tempInvalid,
			warAndPeacePercent: 50,
			responseFormat:     "text",
			responseSchema:     "",
			wantErr:            true,
			errMsg:             "temperature must be between 0 and 2",
		},
		{
			name:               "invalid war and peace percent",
			message:            "Hello",
			temperature:        &temp07,
			warAndPeacePercent: 150,
			responseFormat:     "text",
			responseSchema:     "",
			wantErr:            true,
			errMsg:             "war_and_peace_percent must be between 0 and 100",
		},
		{
			name:               "invalid response format",
			message:            "Hello",
			temperature:        &temp07,
			warAndPeacePercent: 50,
			responseFormat:     "yaml",
			responseSchema:     "",
			wantErr:            true,
			errMsg:             "response_format must be one of: text, json, xml",
		},
		{
			name:               "json format without schema",
			message:            "Hello",
			temperature:        &temp07,
			warAndPeacePercent: 50,
			responseFormat:     "json",
			responseSchema:     "",
			wantErr:            true,
			errMsg:             "response_schema is required for json format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateChatRequest(tt.message, tt.temperature, tt.warAndPeacePercent, tt.responseFormat, tt.responseSchema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateChatRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateChatRequest() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestChatRequestValidator_ValidateSummarizeRequest(t *testing.T) {
	validator := NewChatRequestValidator()

	temp07 := 0.7
	tempInvalid := 3.0

	tests := []struct {
		name        string
		temperature *float64
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid summarize request - nil temperature",
			temperature: nil,
			wantErr:     false,
		},
		{
			name:        "valid summarize request - with temperature",
			temperature: &temp07,
			wantErr:     false,
		},
		{
			name:        "invalid temperature",
			temperature: &tempInvalid,
			wantErr:     true,
			errMsg:      "temperature must be between 0 and 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSummarizeRequest(tt.temperature)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSummarizeRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateSummarizeRequest() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
