# ML Service Visualization Guide

## Overview

The ML service provides comprehensive visualization endpoints for understanding strategy performance, ML model progress, and backtesting results. All endpoints return JSON responses suitable for frontend dashboards and programmatic analysis.

## Base URL

```
http://ml-service:5000
```

## Authentication

All endpoints require no authentication in development. In production, implement Bearer token authentication:

```
Authorization: Bearer <api_token>
```

## Endpoints

### Strategy Ranking - Detailed

**Endpoint:** `GET /visualize/strategy-ranking/detailed`

Get detailed ranking of all trained strategies with comprehensive metrics.

**Query Parameters:**
- `min_confidence` (float, default 0.0) - Minimum ML confidence threshold
- `recommendation` (string, optional) - Filter by recommendation type (DEPLOY, EVALUATE, HOLD)
- `date_range` (string, optional) - Date range for evaluation (e.g., "7d", "30d")

**Response:**
```json
[
  {
    "strategy_id": "strategy_001",
    "strategy_name": "SimpleValueStrategy",
    "composite_score": 0.856,
    "sharpe_ratio": 1.45,
    "roi": 0.125,
    "win_rate": 0.563,
    "max_drawdown": 0.182,
    "profit_factor": 1.89,
    "total_bets": 1250,
    "ml_confidence": 0.92,
    "recommendation": "DEPLOY",
    "backtest_metrics": {
      "historical": {"score": 0.84, "samples": 500},
      "monte_carlo": {"score": 0.86, "samples": 1000},
      "walk_forward": {"score": 0.87, "samples": 250}
    }
  }
]
```

**Example:**
```bash
curl "http://ml-service:5000/visualize/strategy-ranking/detailed?min_confidence=0.8"
```

---

### Strategy Deployment Recommendations

**Endpoint:** `GET /visualize/strategy-deployment-recommendations`

Get strategies recommended for deployment based on backtesting and ML analysis.

**Query Parameters:**
- `risk_level` (string, optional) - Filter by risk level (low, medium, high)
- `target_return` (float, optional) - Minimum target return

**Response:**
```json
[
  {
    "strategy_id": "strategy_001",
    "strategy_name": "SimpleValueStrategy",
    "deployment_confidence": 0.92,
    "expected_sharpe": 1.45,
    "expected_return": 0.125,
    "risk_assessment": {
      "risk_level": "low",
      "max_drawdown": 0.182,
      "volatility": 0.086
    },
    "reasoning": "High composite score, consistent across backtest methods, strong ML confidence"
  }
]
```

**Example:**
```bash
curl "http://ml-service:5000/visualize/strategy-deployment-recommendations?risk_level=low"
```

---

### Feature Importance - Detailed

**Endpoint:** `GET /visualize/feature-importance/{model_name}/detailed`

Get detailed feature importance analysis for a specific ML model.

**Path Parameters:**
- `model_name` (string, required) - Name of the ML model

**Response:**
```json
{
  "model_name": "strategy_predictor_v2",
  "top_features": [
    {
      "feature_name": "rsi_14",
      "importance": 0.234,
      "shap_mean_magnitude": 0.156,
      "trend": "stable"
    },
    {
      "feature_name": "bollinger_width",
      "importance": 0.187,
      "shap_mean_magnitude": 0.142,
      "trend": "increasing"
    }
  ],
  "correlation_matrix": {
    "rsi_14": {"bollinger_width": 0.34, "momentum": 0.12},
    "bollinger_width": {"rsi_14": 0.34, "momentum": 0.45}
  },
  "partial_dependence": {
    "rsi_14": {
      "values": [0.2, 0.4, 0.6, 0.8],
      "predictions": [0.45, 0.62, 0.78, 0.85]
    }
  }
}
```

**Example:**
```bash
curl "http://ml-service:5000/visualize/feature-importance/strategy_predictor_v2/detailed"
```

---

### Strategy Comparison

**Endpoint:** `POST /visualize/compare-strategies`

Compare multiple strategies side-by-side with statistical significance testing.

**Request Body:**
```json
{
  "strategy_ids": ["strategy_001", "strategy_002", "strategy_003"]
}
```

**Response:**
```json
{
  "strategies": [
    {
      "strategy_id": "strategy_001",
      "strategy_name": "SimpleValueStrategy",
      "sharpe_ratio": 1.45,
      "roi": 0.125,
      "win_rate": 0.563,
      "max_drawdown": 0.182,
      "profit_factor": 1.89
    }
  ],
  "statistical_tests": {
    "correlation_matrix": [[1.0, 0.34], [0.34, 1.0]],
    "t_test_results": {
      "strategy_001_vs_strategy_002": {"pvalue": 0.032, "significant": true}
    }
  }
}
```

**Example:**
```bash
curl -X POST "http://ml-service:5000/visualize/compare-strategies" \
  -H "Content-Type: application/json" \
  -d '{"strategy_ids": ["strategy_001", "strategy_002"]}'
```

---

### Backtest Aggregation

**Endpoint:** `GET /visualize/backtest-aggregation/{strategy_id}`

Get aggregated backtest results across all methods for a specific strategy.

**Path Parameters:**
- `strategy_id` (string, required) - ID of the strategy

**Response:**
```json
{
  "strategy_id": "strategy_001",
  "composite_score": 0.856,
  "component_scores": {
    "historical_replay": {
      "score": 0.84,
      "weight": 0.4,
      "weighted_contribution": 0.336,
      "samples": 500
    },
    "monte_carlo": {
      "score": 0.86,
      "weight": 0.35,
      "weighted_contribution": 0.301,
      "samples": 1000
    },
    "walk_forward": {
      "score": 0.87,
      "weight": 0.25,
      "weighted_contribution": 0.2175,
      "samples": 250
    }
  },
  "recommendation": "DEPLOY",
  "reasoning": "Consistent performance across all backtest methods with high confidence"
}
```

**Example:**
```bash
curl "http://ml-service:5000/visualize/backtest-aggregation/strategy_001"
```

---

## Dashboard Endpoints

### Overview

**Endpoint:** `GET /dashboard/overview`

Get summary metrics for the main dashboard.

**Response:**
```json
{
  "bankroll": {
    "current": 10500,
    "initial": 10000,
    "change_percent": 5.0
  },
  "pnl": {
    "daily": 500,
    "weekly": 1200,
    "monthly": 3400
  },
  "active_strategies": 8,
  "circuit_breaker": {
    "status": "NORMAL",
    "trips_today": 0
  },
  "total_bets_placed": 1250,
  "win_rate": 0.563
}
```

---

### Strategy Performance

**Endpoint:** `GET /dashboard/strategy-performance`

Get strategy performance time series data for dashboard visualization.

**Response:**
```json
{
  "timestamp": "2024-02-03T12:00:00Z",
  "strategies": [
    {
      "strategy_id": "strategy_001",
      "name": "SimpleValueStrategy",
      "pnl": 450,
      "active_bets": 12,
      "signals_last_hour": 5,
      "win_rate": 0.58
    }
  ]
}
```

---

### ML Health

**Endpoint:** `GET /dashboard/ml-health`

Get ML service health and performance metrics.

**Response:**
```json
{
  "ml_service": {
    "status": "HEALTHY",
    "uptime_seconds": 86400,
    "last_health_check": "2024-02-03T12:00:00Z"
  },
  "prediction_metrics": {
    "cache_hit_ratio": 0.78,
    "p50_latency_ms": 45,
    "p95_latency_ms": 230,
    "p99_latency_ms": 450,
    "error_rate": 0.002
  },
  "training_jobs": {
    "active": 1,
    "completed_today": 2,
    "failed": 0
  }
}
```

---

### Recent Decisions

**Endpoint:** `GET /dashboard/recent-decisions`

Get recent strategy decisions with context.

**Response:**
```json
{
  "decisions": [
    {
      "timestamp": "2024-02-03T12:00:00Z",
      "strategy_id": "strategy_001",
      "decision": "PLACE_BET",
      "confidence": 0.87,
      "edge_value": 0.045,
      "stake": 100,
      "outcome": "PENDING"
    }
  ]
}
```

---

## Ranking Methodology

The ML service uses a weighted composite score to rank strategies:

```
composite_score = (0.4 * historical_score) 
                + (0.35 * monte_carlo_score) 
                + (0.25 * walk_forward_score)
```

### Scoring Components

**Historical Replay (40% weight)**
- Backtests on historical market data
- Captures actual market conditions
- Best for realistic performance estimates

**Monte Carlo Simulation (35% weight)**
- Simulates market scenarios using resampling
- Tests robustness to market conditions
- Identifies potential failure modes

**Walk Forward Analysis (25% weight)**
- Tests on hold-out periods
- Simulates live trading conditions
- Most realistic performance indicator

### Score Interpretation

- **0.0 - 0.3**: Poor performance, not recommended
- **0.3 - 0.6**: Moderate performance, requires review
- **0.6 - 0.8**: Good performance, candidate for deployment
- **0.8 - 1.0**: Excellent performance, ready for deployment

## Recommendations

The system generates deployment recommendations based on multiple criteria:

### DEPLOY
- Composite score ≥ 0.80
- Confidence ≥ 0.85
- Consistent across all backtest methods
- Acceptable risk/return profile

### EVALUATE
- Composite score 0.60 - 0.80
- Additional data or testing needed
- Consider for limited deployment

### HOLD
- Composite score < 0.60
- Wait for improvement or refinement
- Monitor for changes

## Integration Examples

### Frontend Dashboard (JavaScript)

```javascript
// Fetch strategy ranking
async function loadStrategyRanking() {
  const response = await fetch('/visualize/strategy-ranking/detailed?min_confidence=0.8');
  const strategies = await response.json();
  
  // Display in table
  const table = document.getElementById('strategy-table');
  strategies.forEach(strategy => {
    const row = table.insertRow();
    row.insertCell(0).textContent = strategy.strategy_name;
    row.insertCell(1).textContent = strategy.composite_score.toFixed(3);
    row.insertCell(2).textContent = strategy.recommendation;
  });
}

// Fetch deployment recommendations
async function loadRecommendations() {
  const response = await fetch('/visualize/strategy-deployment-recommendations?risk_level=low');
  const recommendations = await response.json();
  
  return recommendations;
}
```

### Data Analysis (Python)

```python
import requests
import pandas as pd

# Fetch all strategies
url = 'http://ml-service:5000/visualize/strategy-ranking/detailed'
response = requests.get(url)
strategies = response.json()

# Create DataFrame
df = pd.DataFrame(strategies)

# Filter deployed strategies
deployed = df[df['recommendation'] == 'DEPLOY']

# Calculate average metrics
print(deployed[['strategy_name', 'composite_score', 'sharpe_ratio', 'roi']].describe())
```

## Performance Considerations

### Caching

- Strategy ranking cached for 5 minutes
- Feature importance cached for 1 hour
- Backtest results cached for 24 hours
- Dashboard endpoints cached for 1 minute

### Query Optimization

- Use `min_confidence` to filter results
- Use `risk_level` to limit comparisons
- Request only needed date ranges
- Batch multiple queries when possible

## Error Handling

All endpoints return standard HTTP status codes:

- **200 OK** - Successful request
- **400 Bad Request** - Invalid parameters
- **404 Not Found** - Strategy or model not found
- **500 Internal Server Error** - Server error

Error responses include detailed information:

```json
{
  "error": "strategy_not_found",
  "message": "Strategy 'strategy_999' not found in registry",
  "timestamp": "2024-02-03T12:00:00Z"
}
```

## Rate Limiting

Rate limits per IP:
- 1000 requests/minute for read endpoints
- 100 requests/minute for write endpoints

Exceeding limits returns HTTP 429 (Too Many Requests).

## Related Documentation

- [Monitoring Guide](MONITORING.md)
- [Architecture Overview](ARCHITECTURE.md)
- [ML Strategy Guide](ML_STRATEGY.md)
- [Backtesting Guide](BACKTESTING.md)
