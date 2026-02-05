# ML Service API Examples

## Overview

This document provides practical examples for interacting with the Clever Better ML Service API. All examples use `curl` for demonstration.

## Base URL

```
# Local Development
http://localhost:8001

# Production
https://ml-service.clever-better.com
```

## Authentication

```bash
# Set API key (if enabled)
export ML_API_KEY="your-api-key-here"
```

## Health Check

```bash
# Check service health
curl -X GET http://localhost:8001/health

# Response
{
  "status": "healthy",
  "models_loaded": 3,
  "version": "1.0.0"
}
```

## Training Endpoints

### Submit Training Job

```bash
curl -X POST http://localhost:8001/train \
  -H "Content-Type: application/json" \
  -d '{
    "model_type": "race_predictor",
    "features": [
      {
        "distance": 1600,
        "going": "Good",
        "odds": 3.5,
        "jockey_win_rate": 0.25,
        "trainer_win_rate": 0.30,
        "form": "1211"
      },
      {
        "distance": 2000,
        "going": "Soft",
        "odds": 4.2,
        "jockey_win_rate": 0.20,
        "trainer_win_rate": 0.25,
        "form": "2112"
      }
    ],
    "labels": [1.0, 0.0],
    "hyperparameters": {
      "learning_rate": 0.01,
      "epochs": 10,
      "batch_size": 32
    }
  }'

# Response
{
  "job_id": "train-20260215-1234",
  "status": "queued",
  "estimated_duration_seconds": 300
}
```

### Check Training Status

```bash
curl -X GET http://localhost:8001/training/status/train-20260215-1234

# Response (In Progress)
{
  "job_id": "train-20260215-1234",
  "status": "training",
  "progress": 0.45,
  "current_epoch": 5,
  "total_epochs": 10
}

# Response (Completed)
{
  "job_id": "train-20260215-1234",
  "status": "completed",
  "model_id": "model-v1.0.0-20260215",
  "accuracy": 0.87,
  "precision": 0.85,
  "recall": 0.89,
  "f1_score": 0.87
}
```

## Prediction Endpoints

### Single Prediction

```bash
curl -X POST http://localhost:8001/predict \
  -H "Content-Type: application/json" \
  -d '{
    "race_id": "1.198765432",
    "runner_id": "12345678",
    "features": {
      "odds": 3.5,
      "distance": 1600,
      "going": "Good",
      "jockey_win_rate": 0.25,
      "trainer_win_rate": 0.30,
      "age": 4,
      "weight": 126,
      "form": "1211",
      "draw": 5,
      "days_since_last_race": 21
    }
  }'

# Response
{
  "prediction_id": "pred-20260215-5678",
  "confidence": 0.82,
  "probability": 0.75,
  "model_id": "model-v1.0.0-20260215",
  "timestamp": "2026-02-15T14:30:00Z"
}
```

### Batch Prediction

```bash
curl -X POST http://localhost:8001/predict/batch \
  -H "Content-Type: application/json" \
  -d '{
    "predictions": [
      {
        "race_id": "1.198765432",
        "runner_id": "12345678",
        "features": {
          "odds": 3.5,
          "distance": 1600,
          "going": "Good"
        }
      },
      {
        "race_id": "1.198765432",
        "runner_id": "87654321",
        "features": {
          "odds": 4.2,
          "distance": 1600,
          "going": "Good"
        }
      }
    ]
  }'

# Response
{
  "predictions": [
    {
      "runner_id": "12345678",
      "confidence": 0.82,
      "probability": 0.75
    },
    {
      "runner_id": "87654321",
      "confidence": 0.78,
      "probability": 0.68
    }
  ],
  "model_id": "model-v1.0.0-20260215",
  "batch_size": 2
}
```

### Prediction with Model Version

```bash
# Use specific model version
curl -X POST http://localhost:8001/predict?model_id=model-v0.9.5 \
  -H "Content-Type: application/json" \
  -d '{
    "race_id": "1.198765432",
    "runner_id": "12345678",
    "features": {
      "odds": 3.5,
      "distance": 1600
    }
  }'
```

## Model Management

### List Available Models

```bash
curl -X GET http://localhost:8001/api/models

# Response
{
  "models": [
    {
      "model_id": "model-v1.0.0-20260215",
      "version": "1.0.0",
      "created_at": "2026-02-15T10:00:00Z",
      "accuracy": 0.87,
      "status": "active"
    },
    {
      "model_id": "model-v0.9.5-20260201",
      "version": "0.9.5",
      "created_at": "2026-02-01T10:00:00Z",
      "accuracy": 0.85,
      "status": "archived"
    }
  ]
}
```

### Get Latest Model

```bash
curl -X GET http://localhost:8001/api/models/latest

# Response
{
  "model_id": "model-v1.0.0-20260215",
  "version": "1.0.0",
  "created_at": "2026-02-15T10:00:00Z",
  "accuracy": 0.87,
  "features": ["odds", "distance", "going", "jockey_win_rate", "trainer_win_rate"],
  "hyperparameters": {
    "learning_rate": 0.01,
    "epochs": 10
  }
}
```

### Get Model Metrics

```bash
curl -X GET http://localhost:8001/api/models/metrics

# Response
{
  "model-v1.0.0-20260215": {
    "accuracy": 0.87,
    "precision": 0.85,
    "recall": 0.89,
    "f1_score": 0.87,
    "predictions_count": 15432,
    "last_prediction": "2026-02-15T14:30:00Z"
  }
}
```

### Rollback Model

```bash
curl -X POST http://localhost:8001/api/models/rollback \
  -H "Content-Type: application/json" \
  -d '{
    "target_model_id": "model-v0.9.5-20260201"
  }'

# Response
{
  "status": "success",
  "active_model": "model-v0.9.5-20260201",
  "previous_model": "model-v1.0.0-20260215"
}
```

## Feedback and Monitoring

### Submit Prediction Feedback

```bash
curl -X POST http://localhost:8001/feedback \
  -H "Content-Type: application/json" \
  -d '{
    "prediction_id": "pred-20260215-5678",
    "actual_outcome": 1,
    "race_id": "1.198765432",
    "runner_id": "12345678"
  }'

# Response
{
  "status": "success",
  "feedback_id": "fb-20260215-9012"
}
```

### Get Dashboard Data

```bash
curl -X GET http://localhost:8001/api/dashboard/overview

# Response
{
  "total_predictions": 15432,
  "model_accuracy": 0.87,
  "predictions_today": 247,
  "recent_models": [
    {
      "model_id": "model-v1.0.0-20260215",
      "accuracy": 0.87,
      "created_at": "2026-02-15T10:00:00Z"
    }
  ],
  "performance_trend": [
    {"date": "2026-02-01", "accuracy": 0.85},
    {"date": "2026-02-08", "accuracy": 0.86},
    {"date": "2026-02-15", "accuracy": 0.87}
  ]
}
```

### Prometheus Metrics

```bash
# Get Prometheus metrics
curl -X GET http://localhost:8001/metrics

# Response (sample)
# HELP ml_predictions_total Total number of predictions made
# TYPE ml_predictions_total counter
ml_predictions_total{model_id="model-v1.0.0"} 15432

# HELP ml_prediction_duration_seconds Time spent making predictions
# TYPE ml_prediction_duration_seconds histogram
ml_prediction_duration_seconds_bucket{le="0.1"} 12000
ml_prediction_duration_seconds_bucket{le="0.5"} 15000

# HELP ml_model_accuracy Current model accuracy
# TYPE ml_model_accuracy gauge
ml_model_accuracy{model_id="model-v1.0.0"} 0.87
```

## Feature Engineering

### Extract Features

```bash
curl -X POST http://localhost:8001/api/features/extract \
  -H "Content-Type: application/json" \
  -d '{
    "race_data": {
      "distance": 1600,
      "going": "Good",
      "race_class": "1",
      "venue": "Ascot"
    },
    "runner_data": {
      "odds": 3.5,
      "jockey": "Ryan Moore",
      "trainer": "Aidan O'\''Brien",
      "age": 4,
      "form": "1211"
    }
  }'

# Response
{
  "features": {
    "odds": 3.5,
    "log_odds": 1.253,
    "distance": 1600,
    "going_encoded": 2,
    "jockey_win_rate": 0.25,
    "trainer_win_rate": 0.30,
    "recent_wins": 2,
    "recent_places": 4,
    "age_factor": 1.0
  }
}
```

### Get Feature Importance

```bash
curl -X GET http://localhost:8001/api/features/importance

# Response
{
  "feature_importance": {
    "odds": 0.35,
    "distance": 0.25,
    "jockey_win_rate": 0.15,
    "trainer_win_rate": 0.10,
    "going": 0.08,
    "form": 0.07
  }
}
```

## Backtest Integration

### Submit Backtest Results

```bash
curl -X POST http://localhost:8001/api/backtest/submit \
  -H "Content-Type: application/json" \
  -d '{
    "backtest_id": "bt-20260215",
    "results": [
      {
        "race_id": "1.198765432",
        "runner_id": "12345678",
        "predicted_probability": 0.75,
        "actual_outcome": 1,
        "bet_placed": true,
        "stake": 10.0,
        "pnl": 25.0
      }
    ]
  }'

# Response
{
  "status": "success",
  "backtest_id": "bt-20260215",
  "results_count": 1
}
```

## Error Handling

### Invalid Request

```bash
curl -X POST http://localhost:8001/predict \
  -H "Content-Type: application/json" \
  -d '{
    "race_id": "1.198765432"
    # Missing required fields
  }'

# Response (422 Unprocessable Entity)
{
  "detail": [
    {
      "loc": ["body", "runner_id"],
      "msg": "field required",
      "type": "value_error.missing"
    },
    {
      "loc": ["body", "features"],
      "msg": "field required",
      "type": "value_error.missing"
    }
  ]
}
```

### Rate Limiting

```bash
# Too many requests
curl -X POST http://localhost:8001/predict \
  -H "Content-Type: application/json" \
  -d '{ ... }'

# Response (429 Too Many Requests)
{
  "error": "Rate limit exceeded",
  "retry_after": 60
}
```

### Model Not Found

```bash
curl -X POST http://localhost:8001/predict?model_id=nonexistent \
  -H "Content-Type: application/json" \
  -d '{ ... }'

# Response (404 Not Found)
{
  "error": "Model not found",
  "model_id": "nonexistent"
}
```

## WebSocket Streaming (Advanced)

### Stream Predictions

```javascript
// JavaScript example
const ws = new WebSocket('ws://localhost:8001/ws/predictions');

ws.onmessage = (event) => {
  const prediction = JSON.parse(event.data);
  console.log('New prediction:', prediction);
};

ws.send(JSON.stringify({
  action: 'subscribe',
  race_id: '1.198765432'
}));
```

## Integration Examples

### Go Client

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type PredictionRequest struct {
    RaceID   string                 `json:"race_id"`
    RunnerID string                 `json:"runner_id"`
    Features map[string]interface{} `json:"features"`
}

func makePrediction(raceID, runnerID string, features map[string]interface{}) error {
    req := PredictionRequest{
        RaceID:   raceID,
        RunnerID: runnerID,
        Features: features,
    }
    
    body, _ := json.Marshal(req)
    resp, err := http.Post(
        "http://localhost:8001/predict",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Handle response
    return nil
}
```

### Python Client

```python
import requests

def make_prediction(race_id, runner_id, features):
    response = requests.post(
        'http://localhost:8001/predict',
        json={
            'race_id': race_id,
            'runner_id': runner_id,
            'features': features
        }
    )
    
    if response.status_code == 200:
        return response.json()
    else:
        raise Exception(f'Prediction failed: {response.text}')
```

## Best Practices

1. **Always handle errors**: Check status codes and handle error responses
2. **Use batch predictions**: More efficient for multiple predictions
3. **Cache model metadata**: Reduce API calls by caching model info
4. **Submit feedback**: Help improve model accuracy
5. **Monitor rate limits**: Implement backoff strategies
6. **Use specific model versions**: For reproducibility in production

## Rate Limits

- **Predictions**: 100 requests/minute per API key
- **Training**: 10 jobs/hour
- **Feedback**: 1000 submissions/hour

## Support

For API issues or questions:
- Documentation: `/docs` (interactive API docs)
- Health: `/health`
- Metrics: `/metrics`

Last Updated: 2026-02-15
