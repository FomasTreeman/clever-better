# Clever Better ML Service

## Overview

Machine learning service for betting strategy optimization using reinforcement learning, neural classifiers, and ensemble methods. It exposes REST and gRPC interfaces for training, prediction, and visualization, and reads backtest results from TimescaleDB for model training.

## Architecture

- FastAPI for REST endpoints
- gRPC service for Go service integration
- Async SQLAlchemy for database access
- MLflow for experiment tracking and model registry
- Structured logging with structlog

### Models
1. **RL Agent (DQN)** - Learns optimal betting strategies through reinforcement learning
2. **Classifier (TensorFlow)** - Predicts race outcome probabilities with calibration
3. **Ensemble (scikit-learn)** - Combines RF, GB, XGBoost, and LightGBM models

## Setup

```bash
cd ml-service
cp .env.example .env
pip install -r requirements.txt
```

### Environment Variables

```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/clever_better"
export MLFLOW_TRACKING_URI="http://localhost:5000"
export MLFLOW_EXPERIMENT_NAME="clever-better"
```

### Start MLflow Server

```bash
mlflow server --host 0.0.0.0 --port 5000
```

## Run Locally

```bash
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
python -m app.grpc_server
```

## API Endpoints

### Core
- GET /health
- GET /api/v1/health
- GET /api/v1/backtest-results
- GET /api/v1/backtest-results/{id}
- POST /api/v1/preprocess
- POST /api/v1/features/extract
- GET /api/v1/strategies/{strategy_id}/performance
- POST /api/v1/strategies/rank

### Training
- POST /api/v1/models/train
- GET /api/v1/models/training/{job_id}
- GET /api/v1/models/training

### Prediction
- POST /api/v1/predict/race-outcome
- POST /api/v1/predict/strategy-recommendation
- POST /api/v1/predict/ensemble-prediction

### Visualization
- GET /api/v1/visualize/training-progress/{run_id}
- GET /api/v1/visualize/strategy-ranking
- GET /api/v1/visualize/feature-importance/{model_name}

## gRPC

Proto file: proto/ml_service.proto

Generate code:

```bash
make proto-gen
```

## Training Models

```bash
curl -X POST http://localhost:8000/api/v1/models/train \
	-H "Content-Type: application/json" \
	-d '{
		"model_type": "ensemble",
		"hyperparameter_search": true,
		"n_trials": 50
	}'
```

## Testing

```bash
pytest tests/ -v
```

## Documentation

- [Model Architecture](../docs/MODEL_ARCHITECTURE.md)
- [Training Guide](../docs/TRAINING_GUIDE.md)
- [API Examples](../docs/API_EXAMPLES.md)
