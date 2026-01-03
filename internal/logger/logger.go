package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger handles logging to both console and file
type Logger struct {
	file   *os.File
	logger *log.Logger
}

// Config holds logger configuration
type Config struct {
	LogDir     string
	LogFile    string
	MaxSizeMB  int
	ToConsole  bool
}

// DefaultConfig returns default logging configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		LogDir:    filepath.Join(homeDir, ".blocker", "logs"),
		LogFile:   "blocker.log",
		MaxSizeMB: 10,
		ToConsole: true,
	}
}

// New creates a new Logger instance
func New(cfg Config) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(cfg.LogDir, cfg.LogFile)

	// Open log file in append mode
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer for both file and console
	var writer io.Writer
	if cfg.ToConsole {
		writer = io.MultiWriter(os.Stdout, file)
	} else {
		writer = file
	}

	logger := log.New(writer, "", log.LstdFlags)

	// Also set the default logger
	log.SetOutput(writer)
	log.SetFlags(log.LstdFlags)

	return &Logger{
		file:   file,
		logger: logger,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.logger.Printf("[INFO] "+format, args...)
}

// Blocked logs a blocked request
func (l *Logger) Blocked(domain, pattern string) {
	l.logger.Printf("[BLOCKED] %s (matched: %s)", domain, pattern)
}

// Allowed logs an allowed request
func (l *Logger) Allowed(domain string) {
	l.logger.Printf("[ALLOWED] %s", domain)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.logger.Printf("[ERROR] "+format, args...)
}

// GetLogPath returns the path to the log file
func GetLogPath() string {
	cfg := DefaultConfig()
	return filepath.Join(cfg.LogDir, cfg.LogFile)
}

// RotateIfNeeded rotates the log file if it exceeds max size
func (l *Logger) RotateIfNeeded(maxSizeMB int) error {
	if l.file == nil {
		return nil
	}

	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	maxBytes := int64(maxSizeMB * 1024 * 1024)
	if info.Size() < maxBytes {
		return nil
	}

	// Close current file
	l.file.Close()

	// Rename current file with timestamp
	logPath := filepath.Join(filepath.Dir(l.file.Name()), filepath.Base(l.file.Name()))
	backupPath := logPath + "." + time.Now().Format("2006-01-02-150405")
	os.Rename(logPath, backupPath)

	// Open new file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.logger.SetOutput(io.MultiWriter(os.Stdout, file))
	log.SetOutput(io.MultiWriter(os.Stdout, file))

	return nil
}
