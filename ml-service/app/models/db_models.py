from __future__ import annotations

import datetime as dt

from sqlalchemy import DateTime, Float, Index, Integer, String, Text
from sqlalchemy.dialects.postgresql import JSONB, UUID
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    pass


class Strategy(Base):
    __tablename__ = "strategies"

    id: Mapped[str] = mapped_column(UUID(as_uuid=True), primary_key=True)
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    description: Mapped[str | None] = mapped_column(Text, nullable=True)
    parameters: Mapped[dict | None] = mapped_column(JSONB, nullable=True)
    active: Mapped[bool] = mapped_column(nullable=False, default=True)
    created_at: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    updated_at: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), nullable=False)


class BacktestResult(Base):
    __tablename__ = "backtest_results"

    id: Mapped[str] = mapped_column(UUID(as_uuid=True), primary_key=True)
    strategy_id: Mapped[str] = mapped_column(UUID(as_uuid=True), nullable=False, index=True)
    run_date: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), nullable=False, index=True)
    start_date: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    end_date: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    initial_capital: Mapped[float] = mapped_column(Float, nullable=False)
    final_capital: Mapped[float] = mapped_column(Float, nullable=False)
    total_return: Mapped[float] = mapped_column(Float, nullable=False)
    sharpe_ratio: Mapped[float] = mapped_column(Float, nullable=False)
    max_drawdown: Mapped[float] = mapped_column(Float, nullable=False)
    total_bets: Mapped[int] = mapped_column(Integer, nullable=False)
    win_rate: Mapped[float] = mapped_column(Float, nullable=False)
    profit_factor: Mapped[float] = mapped_column(Float, nullable=False)
    method: Mapped[str] = mapped_column(String(64), nullable=False)
    composite_score: Mapped[float] = mapped_column(Float, nullable=False)
    recommendation: Mapped[str] = mapped_column(String(32), nullable=False)
    ml_features: Mapped[dict | None] = mapped_column(JSONB, nullable=True)
    full_results: Mapped[dict | None] = mapped_column(JSONB, nullable=True)
    created_at: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), nullable=False)

    __table_args__ = (
        Index("idx_backtest_results_strategy_id", "strategy_id", "run_date"),
        Index("idx_backtest_results_composite_score", "composite_score"),
    )


class ModelMetadata(Base):
    """Track ML model training runs and versions."""
    __tablename__ = "model_metadata"

    id: Mapped[str] = mapped_column(UUID(as_uuid=True), primary_key=True)
    model_name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    model_type: Mapped[str] = mapped_column(String(64), nullable=False)
    version: Mapped[str] = mapped_column(String(32), nullable=False)
    mlflow_run_id: Mapped[str] = mapped_column(String(255), nullable=False)
    stage: Mapped[str] = mapped_column(String(32), nullable=False, default="None")
    metrics: Mapped[dict | None] = mapped_column(JSONB, nullable=True)
    hyperparameters: Mapped[dict | None] = mapped_column(JSONB, nullable=True)
    feature_names: Mapped[list | None] = mapped_column(JSONB, nullable=True)
    training_dataset_size: Mapped[int | None] = mapped_column(Integer, nullable=True)
    created_at: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    updated_at: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), nullable=False)

    __table_args__ = (
        Index("idx_model_metadata_name_version", "model_name", "version"),
        Index("idx_model_metadata_stage", "stage"),
    )
