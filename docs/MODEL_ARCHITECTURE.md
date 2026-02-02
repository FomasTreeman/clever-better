# Model Architecture

## Overview

The ML service implements a multi-model system for betting strategy optimization:

1. **RL Agent (DQN)** - Learns optimal betting strategies through reinforcement learning
2. **Classifier (TensorFlow)** - Predicts race outcome probabilities with calibration
3. **Ensemble** - Combines multiple tree-based models for robust predictions

## 1. Reinforcement Learning Agent

### Architecture
- **Environment**: `BettingEnvironment`
  - State space: [capital, cumulative_return, Sharpe ratio, max_drawdown, ...market features]
  - Action space: 11 discrete actions (0-100% of Kelly criterion in 10% steps)
  - Reward: `sharpe_ratio * log(profit_factor) - 10 * drawdown_penalty`

- **Network**: `PolicyNetwork`
  - Layer 1: Linear(state_size → 256) + ReLU
  - Layer 2: Linear(256 → 128) + ReLU
  - Layer 3: Linear(128 → 64) + ReLU
  - Output: Linear(64 → action_size)

- **Agent**: `DQNAgent`
  - Experience replay buffer (10,000 transitions)
  - Epsilon-greedy exploration (ε decay 0.995)
  - Target network (updated every 100 steps)
  - Optimizer: Adam (lr=0.001)

### Training
```python
from app.models.rl_agent import train_rl_agent

model, rewards = train_rl_agent(
    episodes=1000,
    initial_capital=10000.0
)
```

## 2. Neural Classifier

### Architecture
- Layer 1: Dense(input_dim → 128) + BatchNorm + Dropout(0.3)
- Layer 2: Dense(128 → 64) + BatchNorm + Dropout(0.2)
- Layer 3: Dense(64 → 32)
- Output: Dense(32 → 1, sigmoid)

### Loss Function
```python
loss = binary_crossentropy + 0.1 * calibration_penalty
```

### Calibration
Post-training isotonic regression for probability calibration

### Training
```python
from app.models.classifier import RaceOutcomeClassifier

classifier = RaceOutcomeClassifier(input_dim=20)
model, history = classifier.train(X_train, y_train, X_val, y_val, epochs=100)
```

## 3. Ensemble Models

### Base Models
1. **RandomForest** (500 trees, max_depth=15)
2. **GradientBoosting** (300 estimators, lr=0.1)
3. **XGBoost** (500 estimators, max_depth=7, lr=0.1)
4. **LightGBM** (500 estimators, num_leaves=31, lr=0.1)

### Ensemble Types
- **Stacking**: LogisticRegression meta-learner
- **Voting**: Soft voting (probability averaging)

### Feature Importance
Uses SHAP values for model interpretability

### Training
```python
from app.models.ensemble import StrategyEnsemble

ensemble = StrategyEnsemble(ensemble_type='stacking')
ensemble.fit(X_train, y_train)
predictions = ensemble.predict_proba(X_test)
```

## Model Selection

### Evaluation Metrics
**Classification**:
- Accuracy, Precision, Recall, F1
- ROC-AUC, Brier score

**Betting**:
- ROI, Sharpe ratio
- Max drawdown, Profit factor, Win rate

### Composite Score
```python
composite = (
    0.30 * sharpe_ratio +
    0.20 * roi +
    0.20 * profit_factor +
    0.15 * (1 - brier_score) +
    0.15 * win_rate
)
```

## Deployment

Models are versioned and tracked with MLflow:
```python
from app.model_registry import ModelRegistry

registry = ModelRegistry(
    tracking_uri="http://localhost:5000",
    experiment_name="clever-better"
)

version = registry.register_model(
    model=trained_model,
    model_name="ensemble",
    model_type="sklearn",
    metrics=evaluation_metrics,
    params=hyperparameters
)
```

Production models are loaded via:
```python
model = registry.get_production_model("ensemble")
```
