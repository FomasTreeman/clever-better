# Training Guide

## Prerequisites

1. **Database Setup**
   - TimescaleDB with backtesting results
   - Required tables: `backtest_results`, `ml_features`

2. **MLflow Setup**
   ```bash
   mlflow server --host 0.0.0.0 --port 5000
   ```

3. **Environment Variables**
   ```bash
   export DATABASE_URL="postgresql://user:pass@localhost:5432/clever_better"
   export MLFLOW_TRACKING_URI="http://localhost:5000"
   export MLFLOW_EXPERIMENT_NAME="clever-better"
   ```

## Training Pipeline

### 1. Data Preparation

The training pipeline automatically loads data from the database:
```python
from app.training import TrainingPipeline, TrainingConfig

config = TrainingConfig(
    database_url="postgresql://...",
    min_composite_score=0.5
)

pipeline = TrainingPipeline(config)
data = await pipeline.load_data_from_database()
```

### 2. Data Preprocessing

Temporal train/val/test split (60%/20%/20%):
```python
X_train, X_val, X_test, y_train, y_val, y_test = pipeline.preprocess_data(data)
```

Features extracted:
- `sharpe_ratio`, `roi`, `max_drawdown`
- `win_rate`, `profit_factor`, `total_bets`
- Additional market features from `ml_features`

### 3. Model Training

#### Train Ensemble (Recommended)
```python
model = await pipeline.train_model(
    X_train, y_train, X_val, y_val,
    model_type='ensemble'
)
```

#### Train Classifier
```python
model = await pipeline.train_model(
    X_train, y_train, X_val, y_val,
    model_type='classifier'
)
```

#### Train RL Agent
```python
model = await pipeline.train_model(
    X_train, y_train, X_val, y_val,
    model_type='rl_agent'
)
```

### 4. Hyperparameter Optimization

Using Optuna for Bayesian optimization:
```python
best_params = await pipeline.hyperparameter_search(
    X_train, y_train, X_val, y_val,
    model_type='ensemble',
    n_trials=100
)
```

Optimizes:
- Number of estimators
- Learning rate
- Max depth
- Regularization parameters

### 5. Model Evaluation

```python
from app.evaluation import ModelEvaluator

evaluator = ModelEvaluator()

# Classification metrics
y_pred = model.predict(X_test)
y_pred_proba = model.predict_proba(X_test)
metrics = evaluator.calculate_classification_metrics(y_test, y_pred, y_pred_proba)

# Betting metrics
stakes = np.array([...])
returns = np.array([...])
betting_metrics = evaluator.calculate_betting_metrics(stakes, returns)
```

### 6. Model Registration

```python
from app.model_registry import ModelRegistry

registry = ModelRegistry(
    tracking_uri="http://localhost:5000",
    experiment_name="clever-better"
)

version = registry.register_model(
    model=model,
    model_name="ensemble",
    model_type="sklearn",
    metrics=metrics,
    params=best_params
)
```

### 7. Model Promotion

```python
# Promote to production
registry.promote_model(
    model_name="ensemble",
    version=version,
    stage="Production"
)
```

## API Training

### Start Training Job
```bash
curl -X POST http://localhost:8000/api/v1/models/train \
  -H "Content-Type: application/json" \
  -d '{
    "model_type": "ensemble",
    "config": {
      "min_composite_score": 0.5
    },
    "hyperparameter_search": true,
    "n_trials": 50
  }'
```

Response:
```json
{
  "job_id": "train_ensemble_1234567890.123",
  "status": "pending",
  "message": "Training job started"
}
```

### Check Training Status
```bash
curl http://localhost:8000/api/v1/models/training/train_ensemble_1234567890.123
```

Response:
```json
{
  "job_id": "train_ensemble_1234567890.123",
  "status": "completed",
  "model_type": "ensemble",
  "created_at": "2024-01-15T10:30:00",
  "completed_at": "2024-01-15T11:45:00",
  "metrics": {
    "accuracy": 0.85,
    "roc_auc": 0.92,
    "sharpe_ratio": 1.8
  },
  "model_version": "3"
}
```

## Incremental Training

Update existing models with new data:
```python
model = await pipeline.incremental_train(
    existing_model=current_model,
    X_new=new_data_X,
    y_new=new_data_y
)
```

## Best Practices

1. **Data Quality**
   - Ensure minimum 1000+ backtesting results
   - Filter low-quality results (composite_score < 0.3)
   - Check for data leakage

2. **Temporal Validation**
   - Always use temporal splits (train on past, validate on future)
   - Implement walk-forward validation for production

3. **Model Selection**
   - Start with ensemble (best out-of-box performance)
   - Use RL agent for dynamic strategy optimization
   - Use classifier for pure probability prediction

4. **Hyperparameter Tuning**
   - Run at least 100 Optuna trials
   - Monitor validation AUC during search
   - Use early stopping to prevent overfitting

5. **Monitoring**
   - Track training metrics in MLflow
   - Monitor composite scores
   - Compare against baseline models

## Troubleshooting

### Low Validation AUC
- Increase model complexity
- Add more features
- Check for class imbalance

### Overfitting
- Increase dropout rates
- Add L2 regularization
- Reduce model complexity

### Poor Betting Metrics
- Adjust reward function in RL agent
- Recalibrate probabilities
- Review betting thresholds
