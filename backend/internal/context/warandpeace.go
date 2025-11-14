package context

import (
	"chat-app/internal/logger"
	"fmt"
	"os"
	"sync"
)

var (
	warAndPeaceText string
	once            sync.Once
)

// LoadWarAndPeace loads the War and Peace text from file
func LoadWarAndPeace(filepath string) error {
	var err error
	once.Do(func() {
		data, readErr := os.ReadFile(filepath)
		if readErr != nil {
			err = fmt.Errorf("error reading War and Peace file: %w", readErr)
			return
		}
		warAndPeaceText = string(data)
		logger.Log.WithField("size_mb", float64(len(data))/1024/1024).Info("Loaded War and Peace text")
	})
	return err
}

// GetWarAndPeace returns the loaded War and Peace text
func GetWarAndPeace() string {
	return warAndPeaceText
}
