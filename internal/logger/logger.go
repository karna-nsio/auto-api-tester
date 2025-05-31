package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger provides logging functionality
type Logger struct {
	*log.Logger
	file *os.File
}

// NewLogger creates a new logger instance
func NewLogger(logDir string) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("llm_%s.log", timestamp))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Create logger
	logger := log.New(file, "", log.LstdFlags)
	return &Logger{
		Logger: logger,
		file:   file,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// LogLLMInteraction logs an LLM interaction
func (l *Logger) LogLLMInteraction(operation string, input interface{}, output interface{}, err error) {
	l.Printf("LLM Operation: %s\n", operation)
	l.Printf("Input: %+v\n", input)
	if err != nil {
		l.Printf("Error: %v\n", err)
	} else {
		l.Printf("Output: %+v\n", output)
	}
	l.Println("---")
}
