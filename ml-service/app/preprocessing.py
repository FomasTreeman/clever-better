from __future__ import annotations

from typing import Any, Dict, Iterable, Optional, Tuple

import pandas as pd
from sqlalchemy import Select, select
from sqlalchemy.ext.asyncio import AsyncSession
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler

from app.models.db_models import BacktestResult


def apply_filters(query: Select, filters: Dict[str, Any]) -> Select:
    if "strategy_id" in filters:
        query = query.where(BacktestResult.strategy_id == filters["strategy_id"])
    if "min_composite_score" in filters:
        query = query.where(BacktestResult.composite_score >= filters["min_composite_score"])
    if "start_date" in filters:
        query = query.where(BacktestResult.run_date >= filters["start_date"])
    if "end_date" in filters:
        query = query.where(BacktestResult.run_date <= filters["end_date"])
    return query


async def load_backtest_results(session: AsyncSession, filters: Dict[str, Any]) -> list[BacktestResult]:
    query = select(BacktestResult)
    query = apply_filters(query, filters)
    result = await session.execute(query)
    return list(result.scalars().all())


def parse_ml_features(backtest_result: BacktestResult) -> Dict[str, Any]:
    return backtest_result.ml_features or {}


def parse_full_results(backtest_result: BacktestResult) -> Dict[str, Any]:
    return backtest_result.full_results or {}


def create_feature_dataframe(backtest_results: Iterable[BacktestResult]) -> pd.DataFrame:
    """Create DataFrame with base metrics and engineered features from full_results."""
    from app.features import create_feature_vector
    
    records = []
    for result in backtest_results:
        base = {
            "id": str(result.id),
            "strategy_id": str(result.strategy_id),
            "run_date": result.run_date,
            "total_return": result.total_return,
            "sharpe_ratio": result.sharpe_ratio,
            "max_drawdown": result.max_drawdown,
            "win_rate": result.win_rate,
            "profit_factor": result.profit_factor,
            "composite_score": result.composite_score,
            "recommendation": result.recommendation,
            "method": result.method,
        }
        base.update(parse_ml_features(result))
        
        # Extract engineered features from full_results
        full_results_dict = dict(result.full_results) if result.full_results else {}
        engineered_features = create_feature_vector({"full_results": full_results_dict})
        
        # Merge numeric engineered features
        for k, v in engineered_features.items():
            if isinstance(v, (int, float)) and k not in base:
                base[k] = v
        
        records.append(base)

    return pd.DataFrame(records)


def apply_feature_engineering(df: pd.DataFrame) -> pd.DataFrame:
    """Apply feature engineering including interaction features.
    
    Adds interaction features on top of the base engineered features already
    extracted in create_feature_dataframe (via create_feature_vector).
    
    Examples of interaction features:
    - odds_vs_form: avg_odds * recent_form
    - trap_grade_interaction: trap_number (as float)
    """
    from app.features import create_interaction_features
    
    if df.empty:
        return df
    
    return create_interaction_features(df)


def handle_missing_values(df: pd.DataFrame) -> pd.DataFrame:
    if df.empty:
        return df
    return df.fillna(0)


def normalize_features(df: pd.DataFrame, scaler: Optional[StandardScaler] = None) -> Tuple[pd.DataFrame, StandardScaler]:
    numeric_cols = df.select_dtypes(include=["number"]).columns
    if scaler is None:
        scaler = StandardScaler()
        df[numeric_cols] = scaler.fit_transform(df[numeric_cols])
    else:
        df[numeric_cols] = scaler.transform(df[numeric_cols])
    return df, scaler


def encode_categorical_features(df: pd.DataFrame) -> pd.DataFrame:
    if df.empty:
        return df
    categorical_cols = ["recommendation", "method"]
    return pd.get_dummies(df, columns=categorical_cols, drop_first=True)


def create_train_test_split(df: pd.DataFrame, test_size: float = 0.2, temporal: bool = True) -> Tuple[pd.DataFrame, pd.DataFrame]:
    if df.empty:
        return df, df
    if temporal and "run_date" in df.columns:
        df = df.sort_values("run_date")
        split_index = int(len(df) * (1 - test_size))
        return df.iloc[:split_index], df.iloc[split_index:]
    train, test = train_test_split(df, test_size=test_size, shuffle=True, random_state=42)
    return train, test
