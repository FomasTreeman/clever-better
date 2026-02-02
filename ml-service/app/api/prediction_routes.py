"""Prediction API endpoints."""
from __future__ import annotations

from typing import List, Dict, Any, Optional
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from app.model_registry import ModelRegistry
from app.config import get_settings
from app import model_io
from app.models.rl_agent import DQNAgent


router = APIRouter(prefix="/predict", tags=["prediction"])
settings = get_settings()


class RaceFeatures(BaseModel):
    features: Dict[str, float]


class PredictionResponse(BaseModel):
    win_probability: float
    place_probability: float
    confidence: float
    recommendation: str


class StrategyRequest(BaseModel):
    race_features: Dict[str, float]
    bankroll: float
    risk_level: str = "medium"


class StrategyResponse(BaseModel):
    recommended_action: str
    stake_size: float
    expected_value: float
    confidence: float


@router.post("/race-outcome", response_model=PredictionResponse)
async def predict_race_outcome(features: RaceFeatures):
    """Predict race outcome probabilities."""
    try:
        # Load production classifier
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name
        )
        
        model = registry.get_production_model("classifier")
        
        # Make prediction
        import numpy as np
        X = np.array([list(features.features.values())])
        prediction = model.predict(X)[0]
        
        # Calculate confidence (placeholder - implement based on your model)
        confidence = 0.85
        
        # Determine recommendation
        if prediction > 0.7:
            recommendation = "strong_bet"
        elif prediction > 0.5:
            recommendation = "moderate_bet"
        else:
            recommendation = "no_bet"
        
        return PredictionResponse(
            win_probability=float(prediction),
            place_probability=float(prediction * 0.8),  # Simplified
            confidence=confidence,
            recommendation=recommendation
        )
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/strategy-recommendation", response_model=StrategyResponse)
async def get_strategy_recommendation(request: StrategyRequest):
    """Get betting strategy recommendation using RL agent."""
    try:
        # Load production RL agent
        agent_path = f"{settings.model_cache_dir}/rl_agent.pt"
        rl_agent = model_io.load_dqn_agent(
            DQNAgent,
            agent_path,
            state_dim=9,
            action_dim=11
        )
        
        # Prepare 9-element state expected by the agent
        import numpy as np
        normalized_capital = 1.0
        state = np.array([
            normalized_capital,
            request.race_features.get('recent_return', 0.0),
            request.race_features.get('sharpe_ratio', 0.0),
            request.race_features.get('max_drawdown', 0.0),
            request.race_features.get('avg_odds', 0.0),
            request.race_features.get('avg_volume', 0.0),
            request.race_features.get('win_rate', 0.0),
            request.race_features.get('profit_factor', 1.0),
            request.race_features.get('var_95', 0.0),
        ], dtype=np.float32)
        
        # Get action
        action = rl_agent.select_action(state, training=False)
        
        # Map action to stake size (0-100% in 10% increments)
        stake_pct = action * 0.1
        stake_size = request.bankroll * stake_pct
        
        # Calculate expected value (placeholder)
        expected_value = stake_size * 0.05
        
        return StrategyResponse(
            recommended_action=f"bet_{int(stake_pct*100)}pct",
            stake_size=stake_size,
            expected_value=expected_value,
            confidence=0.8
        )
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/ensemble-prediction")
async def get_ensemble_prediction(features: RaceFeatures):
    """Get ensemble model prediction."""
    try:
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name
        )
        
        ensemble = registry.get_production_model("ensemble")
        
        import numpy as np
        X = np.array([list(features.features.values())])
        prediction = ensemble.predict_proba(X)[0]
        
        return {
            "prediction": float(prediction),
            "confidence": 0.9
        }
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
