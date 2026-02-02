// Package config provides configuration management for the Clever Better application.
package config

import (
	"fmt"
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
)

// CustomValidator wraps the validator with custom validation rules
type CustomValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new validator with custom validation functions
func NewValidator() *CustomValidator {
	v := validator.New()

	// Register custom validation functions
	v.RegisterValidationFunc("environment", validateEnvironment)
	v.RegisterValidationFunc("loglevel", validateLogLevel)
	v.RegisterValidationFunc("markets", validateMarkets)
	v.RegisterValidationFunc("datetime", validateDateTime)

	return &CustomValidator{validator: v}
}

// Validate validates the entire configuration
func Validate(cfg *Config) error {
	cv := NewValidator()
	return cv.Validate(cfg)
}

// Validate validates the configuration using registered validation rules
func (cv *CustomValidator) Validate(cfg *Config) error {
	err := cv.validator.Struct(cfg)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return formatValidationErrors(validationErrors)
		}
		return fmt.Errorf("validation failed: %w", err)
	}

	// Additional cross-field validations
	if err := validateCrossField(cfg); err != nil {
		return err
	}

	return nil
}

// validateEnvironment validates the environment field
func validateEnvironment(fl validator.FieldLevel) bool {
	env := fl.Field().String()
	switch env {
	case "development", "staging", "production":
		return true
	default:
		return false
	}
}

// validateLogLevel validates the log level field
func validateLogLevel(fl validator.FieldLevel) bool {
	level := fl.Field().String()
	switch level {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

// validateMarkets validates market configuration
func validateMarkets(fl validator.FieldLevel) bool {
	markets := fl.Field().Interface().([]string)
	
	// Check if markets array is not empty
	if len(markets) == 0 {
		return false
	}
	
	validMarkets := map[string]bool{
		"WIN":   true,
		"PLACE": true,
		"EW":    true,
	}

	// Check if all markets are valid
	for _, market := range markets {
		if !validMarkets[market] {
			return false
		}
	}
	return true
}

// validateDateTime validates datetime strings
func validateDateTime(fl validator.FieldLevel) bool {
	dateStr := fl.Field().String()
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

// validateCrossField performs cross-field validations
func validateCrossField(cfg *Config) error {
	// Validate backtest date range
	startDate, err := time.Parse("2006-01-02", cfg.Backtest.StartDate)
	if err != nil {
		return fmt.Errorf("invalid backtest start_date format: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", cfg.Backtest.EndDate)
	if err != nil {
		return fmt.Errorf("invalid backtest end_date format: %w", err)
	}

	if !startDate.Before(endDate) {
		return fmt.Errorf("backtest start_date must be before end_date")
	}

	// Validate production environment requirements
	if cfg.IsProduction() {
		if cfg.Database.SSLMode == "disable" {
			return fmt.Errorf("production environment requires SSL mode to be 'require' or 'verify-full'")
		}
		if !cfg.Features.LiveTradingEnabled && !cfg.Features.PaperTradingEnabled {
			return fmt.Errorf("at least one trading mode must be enabled in production")
		}
	}

	// Validate trading strategy constraints
	if cfg.Trading.MinConfidenceThreshold < 0 || cfg.Trading.MinConfidenceThreshold > 1 {
		return fmt.Errorf("min_confidence_threshold must be between 0 and 1")
	}

	// Validate max daily loss is less than or equal to max exposure
	if cfg.Trading.MaxDailyLoss > cfg.Trading.MaxExposure {
		return fmt.Errorf("max_daily_loss cannot exceed max_exposure")
	}

	// Validate connection pool settings
	if cfg.Database.MaxIdleConnections > cfg.Database.MaxConnections {
		return fmt.Errorf("max_idle_connections cannot exceed max_connections")
	}

	return nil
}

// formatValidationErrors formats validation errors into a readable string
func formatValidationErrors(validationErrors validator.ValidationErrors) error {
	var errMsg string
	for _, fieldError := range validationErrors {
		field := fieldError.StructField()
		tag := fieldError.Tag()
		value := fieldError.Value()

		switch tag {
		case "required":
			errMsg += fmt.Sprintf("- Field '%s' is required\n", field)
		case "url":
			errMsg += fmt.Sprintf("- Field '%s' must be a valid URL, got '%v'\n", field, value)
		case "min", "max":
			errMsg += fmt.Sprintf("- Field '%s' validation failed: %s constraint violated\n", field, tag)
		case "gt", "gte", "lt", "lte":
			errMsg += fmt.Sprintf("- Field '%s' validation failed: numeric constraint %s violated\n", field, tag)
		case "environment":
			errMsg += fmt.Sprintf("- Field '%s' must be one of: development, staging, production\n", field)
		case "loglevel":
			errMsg += fmt.Sprintf("- Field '%s' must be one of: debug, info, warn, error\n", field)
		case "oneof":
			errMsg += fmt.Sprintf("- Field '%s' has invalid value '%v'\n", field, value)
		default:
			errMsg += fmt.Sprintf("- Field '%s' failed validation: %s\n", field, tag)
		}
	}
	return fmt.Errorf("configuration validation failed:\n%s", errMsg)
}

// ValidateEnvironment validates environment-specific requirements
func ValidateEnvironment(cfg *Config) error {
	if cfg.IsProduction() {
		// Production must have SSL enabled
		if cfg.Database.SSLMode == "disable" {
			return fmt.Errorf("production environment requires database SSL mode to be 'require' or 'verify-full'")
		}

		// Production should not have test credentials
		if isTestCredential(cfg.Betfair.Username) {
			return fmt.Errorf("production environment should not use test Betfair credentials")
		}
	}

	if cfg.IsDevelopment() {
		// Development can have permissive settings
		if cfg.Features.LiveTradingEnabled {
			return fmt.Errorf("live trading should be disabled in development mode")
		}
	}

	return nil
}

// isTestCredential checks if a credential looks like a test credential
func isTestCredential(credential string) bool {
	testPatterns := []string{
		"test", "demo", "example", "placeholder", "YOUR_",
	}

	for _, pattern := range testPatterns {
		if match, _ := regexp.MatchString("(?i)"+pattern, credential); match {
			return true
		}
	}

	return false
}
