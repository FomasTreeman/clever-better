// Package logger provides a wrapper around logrus for structured logging.
package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// NewLogger creates a new configured logger instance
func NewLogger(logLevel string) *logrus.Logger {
	logger := logrus.New()
	
	// Set output to stdout
	logger.SetOutput(os.Stdout)
	
	// Parse and set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logger.Warnf("Invalid log level '%s', defaulting to info", logLevel)
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	
	// Use JSON formatter for structured logging in production
	if os.Getenv("ENVIRONMENT") == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		// Use text formatter with colors for development
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}
	
	return logger
}
