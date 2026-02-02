# Clever Better ML Service

## Overview

This service provides REST and gRPC interfaces for ML-based strategy evaluation and feature extraction. It reads backtest results from TimescaleDB and exposes ML-ready data for downstream training.

## Architecture

- FastAPI for REST endpoints
- gRPC service for Go service integration
- Async SQLAlchemy for database access
- Structured logging with structlog

## Setup

```bash
cd ml-service
cp .env.example .env
pip install -r requirements.txt
```

## Run Locally

```bash
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
python -m app.grpc_server
```

## API Endpoints

- GET /health
- GET /api/v1/health
- GET /api/v1/backtest-results
- GET /api/v1/backtest-results/{id}
- POST /api/v1/preprocess
- POST /api/v1/features/extract
- GET /api/v1/strategies/{strategy_id}/performance
- POST /api/v1/strategies/rank

## gRPC

Proto file: proto/ml_service.proto

Generate code:

```bash
make proto-gen
```

## Testing

```bash
pytest tests/ -v
```
