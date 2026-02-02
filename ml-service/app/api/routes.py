from __future__ import annotations

from typing import Any, Dict, List

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.schemas import (
    BacktestResultResponse,
    FeatureExtractionRequest,
    FeatureExtractionResponse,
    HealthCheckResponse,
    StrategyPerformanceResponse,
    StrategyRankingRequest,
    StrategyRankingResponse,
)
from app.database import database_health_check, get_db
from app.features import aggregate_strategy_features, create_feature_vector
from app.models.db_models import BacktestResult
from app.preprocessing import (
        apply_feature_engineering,
    create_feature_dataframe,
    encode_categorical_features,
    handle_missing_values,
    load_backtest_results,
    normalize_features,
)

router = APIRouter(prefix="/api/v1")


@router.get("/health", response_model=HealthCheckResponse)
async def health_check() -> HealthCheckResponse:
    return HealthCheckResponse(status="ok", database=await database_health_check())


@router.get("/backtest-results", response_model=List[BacktestResultResponse])
async def list_backtest_results(
    limit: int = Query(50, ge=1, le=200),
    offset: int = Query(0, ge=0),
    session: AsyncSession = Depends(get_db),
) -> List[BacktestResultResponse]:
    query = select(BacktestResult).order_by(BacktestResult.run_date.desc()).limit(limit).offset(offset)
    results = await session.execute(query)
    records = results.scalars().all()
    return [BacktestResultResponse.model_validate(r, from_attributes=True) for r in records]


@router.get("/backtest-results/{result_id}", response_model=BacktestResultResponse)
async def get_backtest_result(result_id: str, session: AsyncSession = Depends(get_db)) -> BacktestResultResponse:
    result = await session.get(BacktestResult, result_id)
    if not result:
        raise HTTPException(status_code=404, detail="Backtest result not found")
    return BacktestResultResponse.model_validate(result, from_attributes=True)


@router.post("/preprocess", response_model=FeatureExtractionResponse)
async def preprocess_data(payload: FeatureExtractionRequest, session: AsyncSession = Depends(get_db)) -> FeatureExtractionResponse:
    filters = payload.model_dump(exclude_none=True)
    results = await load_backtest_results(session, filters)
    df = create_feature_dataframe(results)
    df = apply_feature_engineering(df)
    df = handle_missing_values(df)
    df = encode_categorical_features(df)
    df, _ = normalize_features(df)
    preview = df.head(20).to_dict(orient="records")
    return FeatureExtractionResponse(rows=len(df), columns=list(df.columns), preview=preview)


@router.post("/features/extract")
async def extract_features(payload: Dict[str, Any]) -> Dict[str, Any]:
    return create_feature_vector(payload)


@router.get("/strategies/{strategy_id}/performance", response_model=StrategyPerformanceResponse)
async def strategy_performance(strategy_id: str, session: AsyncSession = Depends(get_db)) -> StrategyPerformanceResponse:
    query = select(BacktestResult).where(BacktestResult.strategy_id == strategy_id)
    results = await session.execute(query)
    records = results.scalars().all()
    if not records:
        raise HTTPException(status_code=404, detail="Strategy not found")
    features = aggregate_strategy_features([
        {
            "composite_score": r.composite_score,
        }
        for r in records
    ])
    return StrategyPerformanceResponse(
        strategy_id=strategy_id,
        avg_composite_score=features.get("avg_composite_score", 0),
        max_composite_score=features.get("max_composite_score", 0),
        min_composite_score=features.get("min_composite_score", 0),
        total_runs=len(records),
    )


@router.post("/strategies/rank", response_model=StrategyRankingResponse)
async def rank_strategies(payload: StrategyRankingRequest, session: AsyncSession = Depends(get_db)) -> StrategyRankingResponse:
    query = (
        select(
            BacktestResult.strategy_id,
            func.avg(BacktestResult.composite_score).label("avg_score"),
            func.max(BacktestResult.composite_score).label("max_score"),
            func.min(BacktestResult.composite_score).label("min_score"),
            func.count(BacktestResult.id).label("total_runs"),
        )
        .group_by(BacktestResult.strategy_id)
    )
    if payload.min_composite_score is not None:
        query = query.having(func.avg(BacktestResult.composite_score) >= payload.min_composite_score)
    query = query.order_by(func.avg(BacktestResult.composite_score).desc()).limit(payload.limit)

    results = await session.execute(query)
    rankings = [
        StrategyPerformanceResponse(
            strategy_id=row.strategy_id,
            avg_composite_score=float(row.avg_score),
            max_composite_score=float(row.max_score),
            min_composite_score=float(row.min_score),
            total_runs=row.total_runs,
        )
        for row in results.fetchall()
    ]

    return StrategyRankingResponse(rankings=rankings)
