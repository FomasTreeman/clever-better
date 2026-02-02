"""Model evaluation and selection framework."""
from __future__ import annotations

from typing import Any, Dict, List

import numpy as np
from sklearn.metrics import (
    accuracy_score,
    precision_score,
    recall_score,
    f1_score,
    roc_auc_score,
    brier_score_loss,
)


class ModelEvaluator:
    """Comprehensive model assessment and comparison."""
    
    @staticmethod
    def calculate_classification_metrics(
        y_true: np.ndarray,
        y_pred: np.ndarray,
        y_pred_proba: np.ndarray = None
    ) -> Dict[str, float]:
        """Calculate classification metrics."""
        metrics = {
            'accuracy': float(accuracy_score(y_true, y_pred)),
            'precision': float(precision_score(y_true, y_pred, zero_division=0)),
            'recall': float(recall_score(y_true, y_pred, zero_division=0)),
            'f1_score': float(f1_score(y_true, y_pred, zero_division=0)),
        }
        
        if y_pred_proba is not None:
            metrics['roc_auc'] = float(roc_auc_score(y_true, y_pred_proba))
            metrics['brier_score'] = float(brier_score_loss(y_true, y_pred_proba))
        
        return metrics
    
    @staticmethod
    def calculate_betting_metrics(
        returns: np.ndarray,
        win_indicators: np.ndarray
    ) -> Dict[str, float]:
        """Calculate betting-specific metrics."""
        total_return = returns.sum()
        avg_return = returns.mean()
        
        # ROI
        roi = float(total_return / len(returns) if len(returns) > 0 else 0)
        
        # Sharpe ratio
        if returns.std() > 0:
            sharpe_ratio = float(avg_return / returns.std())
        else:
            sharpe_ratio = 0.0
        
        # Max drawdown
        cumulative = np.cumsum(returns)
        running_max = np.maximum.accumulate(cumulative)
        drawdown = running_max - cumulative
        max_drawdown = float(drawdown.max())
        
        # Profit factor
        wins = returns[returns > 0].sum()
        losses = abs(returns[returns < 0].sum())
        profit_factor = float(wins / losses if losses > 0 else 0)
        
        # Win rate
        win_rate = float(win_indicators.mean())
        
        return {
            'roi': roi,
            'sharpe_ratio': sharpe_ratio,
            'max_drawdown': max_drawdown,
            'profit_factor': profit_factor,
            'win_rate': win_rate,
        }
    
    @staticmethod
    def compare_models(
        models_metrics: Dict[str, Dict[str, float]]
    ) -> Dict[str, Any]:
        """Rank models by composite score."""
        composite_scores = {}
        
        for model_name, metrics in models_metrics.items():
            # Composite: 30% Sharpe + 20% ROI + 20% PF + 15% calib + 15% WR
            score = (
                0.30 * metrics.get('sharpe_ratio', 0) +
                0.20 * metrics.get('roi', 0) +
                0.20 * metrics.get('profit_factor', 0) +
                0.15 * (1 - metrics.get('brier_score', 1)) +
                0.15 * metrics.get('win_rate', 0)
            )
            composite_scores[model_name] = float(score)
        
        # Rank models
        ranked = sorted(composite_scores.items(), key=lambda x: x[1], reverse=True)
        
        return {
            'rankings': ranked,
            'best_model': ranked[0][0] if ranked else None,
            'composite_scores': composite_scores,
        }
    
    @staticmethod
    def select_best_model(
        models_metrics: Dict[str, Dict[str, float]]
    ) -> str:
        """Select top performer based on composite score."""
        comparison = ModelEvaluator.compare_models(models_metrics)
        return comparison['best_model']
