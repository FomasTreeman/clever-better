// Package ml provides client interfaces for the ML service.
package ml

import "errors"

var (
	// ErrMLServiceUnavailable indicates the ML service is unreachable
	ErrMLServiceUnavailable = errors.New("ml service unavailable")
	
	// ErrInvalidPrediction indicates the prediction response is invalid
	ErrInvalidPrediction = errors.New("invalid prediction response")
	
	// ErrStrategyGenerationFailed indicates strategy generation failed
	ErrStrategyGenerationFailed = errors.New("strategy generation failed")
	
	// ErrFeedbackSubmissionFailed indicates feedback submission failed
	ErrFeedbackSubmissionFailed = errors.New("feedback submission failed")
	
	// ErrConnectionFailed indicates gRPC connection failed
	ErrConnectionFailed = errors.New("grpc connection failed")
	
	// ErrTimeout indicates request timed out
	ErrTimeout = errors.New("request timeout")
	
	// ErrInvalidResponse indicates invalid response from ML service
	ErrInvalidResponse = errors.New("invalid response from ml service")
)
