"""Visualization API endpoints."""
from __future__ import annotations

from typing import List, Dict, Any
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from app.model_registry import ModelRegistry
from app.config import get_settings


router = APIRouter(prefix="/visualize", tags=["visualization"])
settings = get_settings()


@router.get("/training-progress/{run_id}")
async def get_training_progress(run_id: str):
    """Get training progress metrics."""
    try:
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name
        )
        
        run = registry.client.get_run(run_id)
        metrics = run.data.metrics
        
        return {
            "run_id": run_id,
            "metrics": metrics,
            "status": run.info.status
        }
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/strategy-ranking")
async def get_strategy_ranking():
    """Get ranking of all trained strategies."""
    try:
        from app.evaluation import ModelEvaluator

        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name,
        )

        experiment = registry.client.get_experiment_by_name(settings.mlflow_experiment_name)
        if experiment is None:
            return []

        runs = registry.client.search_runs(experiment.experiment_id)
        models_metrics = {}
        for run in runs:
            model_name = run.data.tags.get("mlflow.runName", run.info.run_id)
            models_metrics[model_name] = run.data.metrics

        evaluator = ModelEvaluator()
        rankings = evaluator.compare_models(models_metrics)
        return rankings

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/feature-importance/{model_name}")
async def get_feature_importance(model_name: str):
    """Get feature importance for a model."""
    try:
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name,
        )

        model = registry.load_model(model_name)

        if hasattr(model, "feature_importances_"):
            importance = model.feature_importances_.tolist()
        else:
            importance = []

        return {
            "model_name": model_name,
            "feature_importance": importance,
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/strategy-ranking/detailed")
async def strategy_ranking_detailed(min_confidence: float = 0.0, recommendation: str = None):
    """Get detailed strategy ranking with comprehensive metrics."""
    try:
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name,
        )

        ranking_data = []
        for model in registry.get_registered_models():
            latest_metrics = registry.get_model_metrics(model.name)
            if latest_metrics.get("confidence", 0) < min_confidence:
                continue

            if recommendation and latest_metrics.get("recommendation") != recommendation:
                continue

            ranking_data.append({
                "strategy_id": model.name,
                "strategy_name": model.name,
                "composite_score": latest_metrics.get("composite_score", 0),
                "sharpe_ratio": latest_metrics.get("sharpe_ratio", 0),
                "roi": latest_metrics.get("roi", 0),
                "win_rate": latest_metrics.get("win_rate", 0),
                "max_drawdown": latest_metrics.get("max_drawdown", 0),
                "profit_factor": latest_metrics.get("profit_factor", 0),
                "total_bets": latest_metrics.get("total_bets", 0),
                "ml_confidence": latest_metrics.get("confidence", 0),
                "recommendation": latest_metrics.get("recommendation", "UNKNOWN"),
            })

        return ranking_data
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/strategy-deployment-recommendations")
async def deployment_recommendations(risk_level: str = None, target_return: float = 0.0):
    """Get strategies recommended for deployment based on backtest results."""
    try:
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name,
        )

        recommendations = []
        for model in registry.get_registered_models():
            metrics = registry.get_model_metrics(model.name)
            if metrics.get("recommendation") != "DEPLOY":
                continue
            if metrics.get("roi", 0) < target_return:
                continue

            recommendations.append({
                "strategy_id": model.name,
                "strategy_name": model.name,
                "deployment_confidence": metrics.get("confidence", 0),
                "expected_sharpe": metrics.get("sharpe_ratio", 0),
                "expected_return": metrics.get("roi", 0),
                "risk_level": risk_level or metrics.get("risk_level", "medium"),
            })

        return recommendations
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/feature-importance/{model_name}/detailed")
async def feature_importance_detailed(model_name: str):
    """Get detailed feature importance for a specific ML model."""
    try:
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name,
        )

        model_metrics = registry.get_model_metrics(model_name)
        return {
            "model_name": model_name,
            "top_features": model_metrics.get("feature_importance", []),
            "correlation_matrix": model_metrics.get("correlations", {}),
            "shap_summary": model_metrics.get("shap_values"),
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/backtest-aggregation/{strategy_id}")
async def backtest_aggregation(strategy_id: str):
    """Get aggregated backtest results from all methods for a strategy."""
    try:
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name,
        )

        metrics = registry.get_model_metrics(strategy_id)
        return {
            "strategy_id": strategy_id,
            "composite_score": metrics.get("composite_score", 0),
            "component_scores": {
                "historical_replay": {
                    "score": metrics.get("historical_score", 0),
                    "weight": 0.4,
                },
                "monte_carlo": {
                    "score": metrics.get("monte_carlo_score", 0),
                    "weight": 0.35,
                },
                "walk_forward": {
                    "score": metrics.get("walk_forward_score", 0),
                    "weight": 0.25,
                },
            },
            "recommendation": metrics.get("recommendation", "UNKNOWN"),
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
