from __future__ import annotations

from datetime import datetime
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field
from pydantic import ConfigDict


class BacktestResultResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)
    id: str
    strategy_id: str
    run_date: datetime
    start_date: datetime
    end_date: datetime
    initial_capital: float
    final_capital: float
    total_return: float
    sharpe_ratio: float
    max_drawdown: float
    total_bets: int
    win_rate: float
    profit_factor: float
    method: str
    composite_score: float
    recommendation: str
    ml_features: Optional[Dict[str, Any]] = None
    full_results: Optional[Dict[str, Any]] = None


class FeatureExtractionRequest(BaseModel):
    strategy_id: Optional[str] = None
    min_composite_score: Optional[float] = None
    start_date: Optional[datetime] = None
    end_date: Optional[datetime] = None


class FeatureExtractionResponse(BaseModel):
    rows: int
    columns: List[str]
    preview: List[Dict[str, Any]]


class StrategyPerformanceResponse(BaseModel):
    strategy_id: str
    avg_composite_score: float
    max_composite_score: float
    min_composite_score: float
    total_runs: int


class StrategyRankingRequest(BaseModel):
    min_composite_score: Optional[float] = Field(default=None)
    limit: int = Field(default=10, ge=1, le=100)


class StrategyRankingResponse(BaseModel):
    rankings: List[StrategyPerformanceResponse]


class HealthCheckResponse(BaseModel):
    status: str
    database: bool


# ML-specific schemas
class TrainingJobRequest(BaseModel):
    model_type: str = Field(description="Model type: rl_agent, classifier, or ensemble")
    config: Optional[Dict[str, Any]] = Field(default=None)
    hyperparameter_search: bool = Field(default=False)
    n_trials: int = Field(default=50, ge=10, le=200)


class TrainingJobResponse(BaseModel):
    job_id: str
    status: str
    message: str


class TrainingStatusResponse(BaseModel):
    job_id: str
    status: str
    model_type: str
    created_at: str
    started_at: Optional[str] = None
    completed_at: Optional[str] = None
    metrics: Optional[Dict[str, float]] = None
    model_version: Optional[str] = None
    error: Optional[str] = None


class PredictionRequest(BaseModel):
    features: Dict[str, float] = Field(description="Feature vector for prediction")


class PredictionResponse(BaseModel):
    win_probability: float
    place_probability: float
    confidence: float
    recommendation: str


class StrategyRecommendationRequest(BaseModel):
    race_features: Dict[str, float]
    bankroll: float
    risk_level: str = Field(default="medium", pattern="^(low|medium|high)$")


class StrategyRecommendationResponse(BaseModel):
    recommended_action: str
    stake_size: float
    expected_value: float
    confidence: float


class ModelMetrics(BaseModel):
    model_name: str
    version: str
    accuracy: Optional[float] = None
    precision: Optional[float] = None
    recall: Optional[float] = None
    f1_score: Optional[float] = None
    roc_auc: Optional[float] = None
    brier_score: Optional[float] = None
    sharpe_ratio: Optional[float] = None
    roi: Optional[float] = None
