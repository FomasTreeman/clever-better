# API Reference

This document provides the API reference for Clever Better services.

## Table of Contents

- [Overview](#overview)
- [ML Service API](#ml-service-api)
- [Internal gRPC API](#internal-grpc-api)
- [Data Models](#data-models)
- [Error Handling](#error-handling)

## Overview

Clever Better exposes two API interfaces:

1. **ML Service REST API**: HTTP/JSON API for ML predictions (FastAPI)
2. **Internal gRPC API**: High-performance binary API for Go-Python communication

## ML Service API

Base URL: `http://localhost:8000` (development) or configured endpoint (production)

### Health Check

Check service health status.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "model_version": "v2.1.0",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Get Prediction

Request a prediction for a specific runner.

**Endpoint:** `POST /api/v1/predict`

**Request Body:**
```json
{
  "race_id": "550e8400-e29b-41d4-a716-446655440000",
  "runner_id": "550e8400-e29b-41d4-a716-446655440001",
  "current_odds": 4.5,
  "features": {
    "trap_number": 1,
    "form_rating": 85.5,
    "days_since_race": 7,
    "track_win_rate": 0.25
  }
}
```

**Response:**
```json
{
  "race_id": "550e8400-e29b-41d4-a716-446655440000",
  "runner_id": "550e8400-e29b-41d4-a716-446655440001",
  "predictions": {
    "win_probability": 0.28,
    "place_probability": 0.55,
    "expected_value": 0.12
  },
  "confidence": 0.85,
  "model_version": "v2.1.0",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Batch Prediction

Request predictions for all runners in a race.

**Endpoint:** `POST /api/v1/predict/batch`

**Request Body:**
```json
{
  "race_id": "550e8400-e29b-41d4-a716-446655440000",
  "runners": [
    {
      "runner_id": "550e8400-e29b-41d4-a716-446655440001",
      "current_odds": 4.5,
      "features": { ... }
    },
    {
      "runner_id": "550e8400-e29b-41d4-a716-446655440002",
      "current_odds": 3.2,
      "features": { ... }
    }
  ]
}
```

**Response:**
```json
{
  "race_id": "550e8400-e29b-41d4-a716-446655440000",
  "predictions": [
    {
      "runner_id": "550e8400-e29b-41d4-a716-446655440001",
      "win_probability": 0.28,
      "place_probability": 0.55,
      "expected_value": 0.12,
      "confidence": 0.85
    },
    {
      "runner_id": "550e8400-e29b-41d4-a716-446655440002",
      "win_probability": 0.35,
      "place_probability": 0.62,
      "expected_value": 0.08,
      "confidence": 0.82
    }
  ],
  "model_version": "v2.1.0",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Get Model Info

Retrieve current model information.

**Endpoint:** `GET /api/v1/model/info`

**Response:**
```json
{
  "model_id": "ensemble-v2.1.0",
  "version": "v2.1.0",
  "trained_at": "2024-01-10T00:00:00Z",
  "training_samples": 1500000,
  "features": [
    "trap_number",
    "form_rating",
    "days_since_race",
    "track_win_rate",
    "..."
  ],
  "metrics": {
    "accuracy": 0.42,
    "brier_score": 0.21,
    "log_loss": 0.65,
    "auc_roc": 0.72
  }
}
```

### Feature Importance

Get feature importance rankings.

**Endpoint:** `GET /api/v1/model/features`

**Response:**
```json
{
  "model_version": "v2.1.0",
  "features": [
    { "name": "current_odds", "importance": 0.18 },
    { "name": "form_rating", "importance": 0.15 },
    { "name": "trap_number", "importance": 0.12 },
    { "name": "days_since_race", "importance": 0.09 },
    { "name": "track_win_rate", "importance": 0.08 }
  ]
}
```

## Internal gRPC API

The gRPC API provides high-performance communication between Go services and the Python ML service.

### Proto Definition

```protobuf
syntax = "proto3";

package cleverbet.ml.v1;

service PredictionService {
  // Get prediction for a single runner
  rpc Predict(PredictRequest) returns (PredictResponse);

  // Get predictions for all runners in a race
  rpc PredictBatch(BatchPredictRequest) returns (BatchPredictResponse);

  // Stream predictions for multiple races
  rpc PredictStream(stream PredictRequest) returns (stream PredictResponse);
}

message PredictRequest {
  string race_id = 1;
  string runner_id = 2;
  double current_odds = 3;
  map<string, double> features = 4;
}

message PredictResponse {
  string race_id = 1;
  string runner_id = 2;
  double win_probability = 3;
  double place_probability = 4;
  double expected_value = 5;
  double confidence = 6;
  string model_version = 7;
  int64 timestamp_ms = 8;
}

message BatchPredictRequest {
  string race_id = 1;
  repeated RunnerFeatures runners = 2;
}

message RunnerFeatures {
  string runner_id = 1;
  double current_odds = 2;
  map<string, double> features = 3;
}

message BatchPredictResponse {
  string race_id = 1;
  repeated PredictResponse predictions = 2;
}
```

### Go Client Usage

```go
package ml

import (
    "context"
    "time"

    pb "github.com/yourusername/clever-better/api/proto/ml/v1"
    "google.golang.org/grpc"
)

type Client struct {
    conn   *grpc.ClientConn
    client pb.PredictionServiceClient
}

func NewClient(addr string) (*Client, error) {
    conn, err := grpc.Dial(addr, grpc.WithInsecure())
    if err != nil {
        return nil, err
    }

    return &Client{
        conn:   conn,
        client: pb.NewPredictionServiceClient(conn),
    }, nil
}

func (c *Client) Predict(ctx context.Context, raceID, runnerID string, odds float64, features map[string]float64) (*pb.PredictResponse, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req := &pb.PredictRequest{
        RaceId:      raceID,
        RunnerId:    runnerID,
        CurrentOdds: odds,
        Features:    features,
    }

    return c.client.Predict(ctx, req)
}
```

## Data Models

### Race

```json
{
  "id": "uuid",
  "scheduled_start": "2024-01-15T14:30:00Z",
  "track": "string",
  "race_type": "string",
  "distance": 480,
  "grade": "A3",
  "prize_money": 500.00,
  "conditions": "string",
  "status": "scheduled|running|completed|abandoned"
}
```

### Runner

```json
{
  "id": "uuid",
  "race_id": "uuid",
  "trap_number": 1,
  "name": "string",
  "trainer": "string",
  "form": "111234",
  "metadata": {
    "weight": 32.5,
    "age": "3y",
    "sex": "D"
  }
}
```

### Prediction

```json
{
  "id": "uuid",
  "race_id": "uuid",
  "runner_id": "uuid",
  "model_id": "uuid",
  "win_probability": 0.28,
  "place_probability": 0.55,
  "expected_value": 0.12,
  "confidence": 0.85,
  "features_hash": "sha256:...",
  "predicted_at": "2024-01-15T14:25:00Z"
}
```

### Trade

```json
{
  "id": "uuid",
  "race_id": "uuid",
  "runner_id": "uuid",
  "strategy_id": "uuid",
  "side": "BACK|LAY",
  "odds": 4.5,
  "stake": 10.00,
  "status": "pending|matched|partially_matched|cancelled|settled",
  "profit_loss": 35.00,
  "betfair_bet_id": "string",
  "executed_at": "2024-01-15T14:28:00Z",
  "settled_at": "2024-01-15T14:35:00Z"
}
```

## Error Handling

### HTTP Error Responses

All errors follow a consistent format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "field": "additional context"
    }
  },
  "request_id": "uuid"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request body |
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `NOT_FOUND` | 404 | Resource not found |
| `RACE_NOT_FOUND` | 404 | Race ID not found |
| `RUNNER_NOT_FOUND` | 404 | Runner ID not found |
| `MODEL_NOT_READY` | 503 | Model not loaded/ready |
| `PREDICTION_FAILED` | 500 | Prediction computation failed |
| `INTERNAL_ERROR` | 500 | Unexpected internal error |

### gRPC Error Codes

| gRPC Code | Description |
|-----------|-------------|
| `INVALID_ARGUMENT` | Invalid request parameters |
| `NOT_FOUND` | Resource not found |
| `UNAVAILABLE` | Service temporarily unavailable |
| `DEADLINE_EXCEEDED` | Request timeout |
| `INTERNAL` | Internal server error |

### Rate Limiting

The API implements rate limiting:

- **Standard tier**: 100 requests/minute
- **Burst**: 20 requests/second

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705315800
```

When rate limited, the API returns:
```json
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded. Retry after 60 seconds.",
    "details": {
      "retry_after": 60
    }
  }
}
```
