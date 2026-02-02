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
        
        # Get all registered models
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name
        )
        
        # Fetch metrics from MLflow
        experiment = registry.client.get_experiment_by_name(settings.mlflow_experiment_name)
        runs = registry.client.search_runs(experiment.experiment_id)
        
        models_metrics = {}
        for run in runs:
            model_name = run.data.tags.get('mlflow.runName', run.info.run_id)
            models_metrics[model_name] = run.data.metrics
        
        # Rank models
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
            experiment_name=settings.mlflow_experiment_name
        )
        
        model = registry.load_model(model_name)
        
        # Get feature importance (implementation depends on model type)
        if hasattr(model, 'feature_importances_'):
            importance = model.feature_importances_.tolist()
        else:
            importance = []
        
        return {
            "model_name": model_name,
            "feature_importance": importance
        }
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
