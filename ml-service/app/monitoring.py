"""Monitoring and logging for ML operations."""
from __future__ import annotations

import logging
from datetime import datetime
from typing import Dict, Any, Optional
import json


class MLLogger:
    """Structured logging for ML operations."""
    
    def __init__(self, name: str):
        self.logger = logging.getLogger(name)
        self.logger.setLevel(logging.INFO)
        
        handler = logging.StreamHandler()
        formatter = logging.Formatter(
            '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        handler.setFormatter(formatter)
        self.logger.addHandler(handler)
    
    def log_prediction(
        self,
        model_name: str,
        features: Dict[str, Any],
        prediction: Any,
        confidence: float,
        latency_ms: float
    ):
        """Log prediction event."""
        self.logger.info(json.dumps({
            'event': 'prediction',
            'timestamp': datetime.utcnow().isoformat(),
            'model_name': model_name,
            'prediction': prediction,
            'confidence': confidence,
            'latency_ms': latency_ms,
            'feature_count': len(features)
        }))
    
    def log_training(
        self,
        model_name: str,
        metrics: Dict[str, float],
        duration_seconds: float,
        status: str
    ):
        """Log training event."""
        self.logger.info(json.dumps({
            'event': 'training',
            'timestamp': datetime.utcnow().isoformat(),
            'model_name': model_name,
            'metrics': metrics,
            'duration_seconds': duration_seconds,
            'status': status
        }))
    
    def log_model_registration(
        self,
        model_name: str,
        version: str,
        metrics: Dict[str, float]
    ):
        """Log model registration."""
        self.logger.info(json.dumps({
            'event': 'model_registration',
            'timestamp': datetime.utcnow().isoformat(),
            'model_name': model_name,
            'version': version,
            'metrics': metrics
        }))
    
    def log_error(
        self,
        operation: str,
        error: str,
        context: Optional[Dict[str, Any]] = None
    ):
        """Log error event."""
        self.logger.error(json.dumps({
            'event': 'error',
            'timestamp': datetime.utcnow().isoformat(),
            'operation': operation,
            'error': error,
            'context': context or {}
        }))


# Global logger instance
ml_logger = MLLogger('ml-service')
