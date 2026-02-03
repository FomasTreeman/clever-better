// Package ml provides HTTP client for ML service batch operations.
package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/yourusername/clever-better/internal/config"
)

// HTTPClient provides HTTP client for ML service
type HTTPClient struct {
	client  *http.Client
	baseURL string
	logger  *logrus.Logger
}

// NewHTTPClient creates a new HTTP client for ML service
func NewHTTPClient(cfg *config.MLServiceConfig, logger *logrus.Logger) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeoutSeconds) * time.Second,
		},
		baseURL: cfg.HTTPAddress,
		logger:  logger,
	}
}

// TrainModelsRequest represents training request payload
type TrainModelsRequest struct {
	ModelType            string            `json:"model_type"`
	Epochs               int               `json:"epochs"`
	BatchSize            int               `json:"batch_size"`
	LearningRate         float64           `json:"learning_rate"`
	HyperparameterSearch bool              `json:"hyperparameter_search"`
	DataFilters          map[string]string `json:"data_filters,omitempty"`
}

// TrainModelsResponse represents training response
type TrainModelsResponse struct {
	JobID      string    `json:"job_id"`
	Status     string    `json:"status"`
	ModelType  string    `json:"model_type"`
	Message    string    `json:"message"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// TrainModels initiates model training via HTTP
func (c *HTTPClient) TrainModels(ctx context.Context, config TrainingConfig) (*TrainingStatus, error) {
	start := time.Now()

	reqBody := TrainModelsRequest{
		ModelType:            config.ModelType,
		Epochs:               config.Epochs,
		BatchSize:            config.BatchSize,
		LearningRate:         config.LearningRate,
		HyperparameterSearch: config.HyperparameterSearch,
		DataFilters:          config.DataFilters,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/models/train", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		MLGRPCErrorsTotal.WithLabelValues("train_models", "network").Inc()
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		MLGRPCErrorsTotal.WithLabelValues("train_models", "http_error").Inc()
		return nil, fmt.Errorf("training request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var trainResp TrainModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&trainResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"job_id":     trainResp.JobID,
		"model_type": trainResp.ModelType,
		"duration":   time.Since(start),
	}).Info("Training job submitted")

	MLTrainingJobsTotal.WithLabelValues(trainResp.ModelType, "submitted").Inc()

	submittedAt := trainResp.SubmittedAt
	return &TrainingStatus{
		JobID:       trainResp.JobID,
		Status:      trainResp.Status,
		ModelType:   trainResp.ModelType,
		SubmittedAt: &submittedAt,
	}, nil
}

// GetTrainingStatus retrieves training job status
func (c *HTTPClient) GetTrainingStatus(ctx context.Context, jobID string) (*TrainingStatus, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/models/train/%s/status", c.baseURL, jobID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status request failed with status %d", resp.StatusCode)
	}

	var status TrainingStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status: %w", err)
	}

	return &status, nil
}

// GetModelMetrics retrieves metrics for a trained model
func (c *HTTPClient) GetModelMetrics(ctx context.Context, modelType string, version int) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/models/%s/metrics?version=%d", c.baseURL, modelType, version), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics request failed with status %d", resp.StatusCode)
	}

	var metrics map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode metrics: %w", err)
	}

	return metrics, nil
}

// HealthCheck checks ML service health
func (c *HTTPClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMLServiceUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrMLServiceUnavailable, resp.StatusCode)
	}

	return nil
}
