# API Examples

## Training API

### 1. Train Ensemble Model

**Request:**
```bash
curl -X POST http://localhost:8000/api/v1/models/train \
  -H "Content-Type: application/json" \
  -d '{
    "model_type": "ensemble",
    "config": {
      "min_composite_score": 0.5,
      "ensemble_type": "stacking"
    },
    "hyperparameter_search": true,
    "n_trials": 100
  }'
```

**Response:**
```json
{
  "job_id": "train_ensemble_1705315800.0",
  "status": "pending",
  "message": "Training job train_ensemble_1705315800.0 started"
}
```

### 2. Check Training Status

**Request:**
```bash
curl http://localhost:8000/api/v1/models/training/train_ensemble_1705315800.0
```

**Response:**
```json
{
  "job_id": "train_ensemble_1705315800.0",
  "status": "completed",
  "model_type": "ensemble",
  "created_at": "2024-01-15T10:30:00.000000",
  "started_at": "2024-01-15T10:30:05.123456",
  "completed_at": "2024-01-15T11:45:30.789012",
  "metrics": {
    "accuracy": 0.847,
    "precision": 0.823,
    "recall": 0.891,
    "f1_score": 0.856,
    "roc_auc": 0.924,
    "brier_score": 0.156
  },
  "model_version": "5",
  "best_params": {
    "n_estimators": 500,
    "max_depth": 12,
    "learning_rate": 0.08
  }
}
```

### 3. List All Training Jobs

**Request:**
```bash
curl http://localhost:8000/api/v1/models/training
```

**Response:**
```json
[
  {
    "job_id": "train_ensemble_1705315800.0",
    "status": "completed",
    "model_type": "ensemble",
    "created_at": "2024-01-15T10:30:00.000000"
  },
  {
    "job_id": "train_classifier_1705320400.0",
    "status": "running",
    "model_type": "classifier",
    "created_at": "2024-01-15T12:00:00.000000"
  }
]
```

## Prediction API

### 1. Predict Race Outcome

**Request:**
```bash
curl -X POST http://localhost:8000/api/v1/predict/race-outcome \
  -H "Content-Type: application/json" \
  -d '{
    "features": {
      "odds": 3.5,
      "form_rating": 0.82,
      "track_condition": 1.0,
      "distance": 1200,
      "weight": 58.5,
      "barrier": 4,
      "jockey_rating": 0.75,
      "trainer_rating": 0.68
    }
  }'
```

**Response:**
```json
{
  "win_probability": 0.723,
  "place_probability": 0.578,
  "confidence": 0.85,
  "recommendation": "strong_bet"
}
```

### 2. Get Strategy Recommendation

**Request:**
```bash
curl -X POST http://localhost:8000/api/v1/predict/strategy-recommendation \
  -H "Content-Type: application/json" \
  -d '{
    "race_features": {
      "sharpe_ratio": 1.8,
      "roi": 0.15,
      "max_drawdown": 0.18,
      "win_rate": 0.58,
      "profit_factor": 1.92
    },
    "bankroll": 10000.0,
    "risk_level": "medium"
  }'
```

**Response:**
```json
{
  "recommended_action": "bet_40pct",
  "stake_size": 4000.0,
  "expected_value": 200.0,
  "confidence": 0.82
}
```

### 3. Ensemble Prediction

**Request:**
```bash
curl -X POST http://localhost:8000/api/v1/predict/ensemble-prediction \
  -H "Content-Type: application/json" \
  -d '{
    "features": {
      "feature1": 0.65,
      "feature2": 0.82,
      "feature3": 0.45,
      "feature4": 0.91
    }
  }'
```

**Response:**
```json
{
  "prediction": 0.768,
  "confidence": 0.9
}
```

## Visualization API

### 1. Get Training Progress

**Request:**
```bash
curl http://localhost:8000/api/v1/visualize/training-progress/run_abc123
```

**Response:**
```json
{
  "run_id": "run_abc123",
  "metrics": {
    "train_loss": 0.234,
    "val_loss": 0.267,
    "train_auc": 0.912,
    "val_auc": 0.898
  },
  "status": "FINISHED"
}
```

### 2. Get Strategy Ranking

**Request:**
```bash
curl http://localhost:8000/api/v1/visualize/strategy-ranking
```

**Response:**
```json
{
  "rankings": [
    {
      "model_name": "ensemble_v5",
      "composite_score": 0.847,
      "rank": 1
    },
    {
      "model_name": "rl_agent_v3",
      "composite_score": 0.821,
      "rank": 2
    },
    {
      "model_name": "classifier_v8",
      "composite_score": 0.789,
      "rank": 3
    }
  ],
  "best_model": "ensemble_v5"
}
```

### 3. Get Feature Importance

**Request:**
```bash
curl http://localhost:8000/api/v1/visualize/feature-importance/ensemble
```

**Response:**
```json
{
  "model_name": "ensemble",
  "feature_importance": [
    {"feature": "odds", "importance": 0.285},
    {"feature": "form_rating", "importance": 0.192},
    {"feature": "track_condition", "importance": 0.156},
    {"feature": "jockey_rating", "importance": 0.134},
    {"feature": "distance", "importance": 0.098},
    {"feature": "weight", "importance": 0.075},
    {"feature": "barrier", "importance": 0.060}
  ]
}
```

## Python SDK Examples

### Training
```python
import requests

# Start training
response = requests.post(
    "http://localhost:8000/api/v1/models/train",
    json={
        "model_type": "ensemble",
        "hyperparameter_search": True,
        "n_trials": 50
    }
)
job_id = response.json()["job_id"]

# Poll for completion
import time
while True:
    status = requests.get(f"http://localhost:8000/api/v1/models/training/{job_id}")
    if status.json()["status"] == "completed":
        break
    time.sleep(10)

print(f"Training complete! Metrics: {status.json()['metrics']}")
```

### Prediction
```python
import requests

# Get prediction
response = requests.post(
    "http://localhost:8000/api/v1/predict/race-outcome",
    json={
        "features": {
            "odds": 3.5,
            "form_rating": 0.82,
            "track_condition": 1.0
        }
    }
)

result = response.json()
print(f"Win probability: {result['win_probability']:.2%}")
print(f"Recommendation: {result['recommendation']}")
```
