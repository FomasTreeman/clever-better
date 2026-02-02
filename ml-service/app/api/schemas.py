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
