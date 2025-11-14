package validation

import (
	"errors"
	"fmt"
)

// ChatRequestValidator validates chat-related requests
type ChatRequestValidator struct{}

// NewChatRequestValidator creates a new ChatRequestValidator
func NewChatRequestValidator() *ChatRequestValidator {
	return &ChatRequestValidator{}
}

// ValidateMessage validates a chat message
func (v *ChatRequestValidator) ValidateMessage(message string) error {
	if message == "" {
		return errors.New("message cannot be empty")
	}
	return nil
}

// ValidateWarAndPeacePercent validates the War and Peace percentage
func (v *ChatRequestValidator) ValidateWarAndPeacePercent(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("war_and_peace_percent must be between 0 and 100, got %d", percent)
	}
	return nil
}

// ValidateTemperature validates the temperature parameter
func (v *ChatRequestValidator) ValidateTemperature(temperature *float64) error {
	if temperature == nil {
		return nil // Temperature is optional
	}

	if *temperature < 0 || *temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2, got %.2f", *temperature)
	}
	return nil
}

// ValidateResponseFormat validates the response format
func (v *ChatRequestValidator) ValidateResponseFormat(format string) error {
	if format == "" {
		return nil // Format is optional, defaults to "text"
	}

	validFormats := map[string]bool{
		"text": true,
		"json": true,
		"xml":  true,
	}

	if !validFormats[format] {
		return fmt.Errorf("response_format must be one of: text, json, xml; got %s", format)
	}
	return nil
}

// ValidateResponseSchema validates that a schema is provided for structured formats
func (v *ChatRequestValidator) ValidateResponseSchema(format, schema string) error {
	if format == "" || format == "text" {
		return nil // Schema not required for text format
	}

	if schema == "" {
		return fmt.Errorf("response_schema is required for %s format", format)
	}
	return nil
}

// ValidateChatRequest validates a complete chat request
func (v *ChatRequestValidator) ValidateChatRequest(message string, temperature *float64, warAndPeacePercent int, responseFormat, responseSchema string) error {
	if err := v.ValidateMessage(message); err != nil {
		return err
	}

	if err := v.ValidateTemperature(temperature); err != nil {
		return err
	}

	if err := v.ValidateWarAndPeacePercent(warAndPeacePercent); err != nil {
		return err
	}

	if err := v.ValidateResponseFormat(responseFormat); err != nil {
		return err
	}

	if err := v.ValidateResponseSchema(responseFormat, responseSchema); err != nil {
		return err
	}

	return nil
}

// ValidateSummarizeRequest validates a summarization request
func (v *ChatRequestValidator) ValidateSummarizeRequest(temperature *float64) error {
	return v.ValidateTemperature(temperature)
}
