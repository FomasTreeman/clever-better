package betfair

import (
	"fmt"
	"log"
)

// BetfairAPIError represents an error from Betfair API
type BetfairAPIError struct {
	Message    string
	ErrorCode  string
	Data       string
	Cause      error
}

func (e *BetfairAPIError) Error() string {
	return fmt.Sprintf("Betfair API error: %s (code: %s)", e.Message, e.ErrorCode)
}

// AuthenticationError represents an authentication failure
type AuthenticationError struct {
	Message string
	Cause   error
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("Authentication error: %s", e.Message)
}

// InsufficientFundsError represents insufficient account funds
type InsufficientFundsError struct {
	Message string
	Cause   error
}

func (e *InsufficientFundsError) Error() string {
	return fmt.Sprintf("Insufficient funds: %s", e.Message)
}

// MarketSuspendedError represents a suspended market
type MarketSuspendedError struct {
	MarketID string
	Message  string
	Cause    error
}

func (e *MarketSuspendedError) Error() string {
	return fmt.Sprintf("Market suspended [%s]: %s", e.MarketID, e.Message)
}

// OrderLimitExceededError represents exceeded order limit
type OrderLimitExceededError struct {
	Message string
	Cause   error
}

func (e *OrderLimitExceededError) Error() string {
	return fmt.Sprintf("Order limit exceeded: %s", e.Message)
}

// NewBetfairAPIError creates a new Betfair API error
func NewBetfairAPIError(message, code string, cause error) *BetfairAPIError {
	return &BetfairAPIError{
		Message:   message,
		ErrorCode: code,
		Cause:     cause,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string, cause error) *AuthenticationError {
	return &AuthenticationError{
		Message: message,
		Cause:   cause,
	}
}

// NewInsufficientFundsError creates a new insufficient funds error
func NewInsufficientFundsError(message string, cause error) *InsufficientFundsError {
	return &InsufficientFundsError{
		Message: message,
		Cause:   cause,
	}
}

// NewMarketSuspendedError creates a new market suspended error
func NewMarketSuspendedError(marketID, message string, cause error) *MarketSuspendedError {
	return &MarketSuspendedError{
		MarketID: marketID,
		Message:  message,
		Cause:    cause,
	}
}

// NewOrderLimitExceededError creates a new order limit exceeded error
func NewOrderLimitExceededError(message string, cause error) *OrderLimitExceededError {
	return &OrderLimitExceededError{
		Message: message,
		Cause:   cause,
	}
}

// MapBetfairError maps Betfair API error codes to specific error types
func MapBetfairError(errorCode string, message string, logger *log.Logger) error {
	if logger != nil {
		logger.Printf("Betfair error code: %s, message: %s", errorCode, message)
	}

	switch errorCode {
	case ErrorInvalidSessionInformation:
		return NewAuthenticationError("Invalid session information", nil)
	case ErrorInsufficientFunds:
		return NewInsufficientFundsError("Insufficient funds for this bet", nil)
	case ErrorMarketSuspended:
		return NewMarketSuspendedError("", "Market suspended", nil)
	case ErrorOrderLimitExceeded:
		return NewOrderLimitExceededError("Order limit has been exceeded", nil)
	case ErrorPersistenceQuotaExceeded:
		return fmt.Errorf("persistence quota exceeded: %s", message)
	case ErrorInvalidBetSize:
		return fmt.Errorf("invalid bet size: %s", message)
	case ErrorOperationNotAllowed:
		return fmt.Errorf("operation not allowed: %s", message)
	default:
		return NewBetfairAPIError(message, errorCode, nil)
	}
}
