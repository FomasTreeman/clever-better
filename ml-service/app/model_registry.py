"""MLflow-based model versioning and registry."""
from __future__ import annotations

from datetime import datetime
from typing import Any, Dict, Optional

import mlflow
from mlflow.tracking import MlflowClient


class ModelRegistry:
    """MLflow model versioning and registry management."""
    
    def __init__(self, tracking_uri: str, experiment_name: str):
        self.tracking_uri = tracking_uri
        self.experiment_name = experiment_name
        mlflow.set_tracking_uri(tracking_uri)
        mlflow.set_experiment(experiment_name)
        self.client = MlflowClient(tracking_uri)
    
    def register_model(
        self,
        model: Any,
        model_name: str,
        model_type: str,
        metrics: Dict[str, float],
        params: Dict[str, Any],
        artifacts: Dict[str, str] = None
    ) -> str:
        """Register model with MLflow."""
        with mlflow.start_run() as run:
            # Log parameters
            mlflow.log_params(params)
            
            # Log metrics
            mlflow.log_metrics(metrics)
            
            # Log model
            if model_type == 'pytorch':
                mlflow.pytorch.log_model(model, "model")
            elif model_type == 'tensorflow':
                mlflow.tensorflow.log_model(model, "model")
            elif model_type == 'sklearn':
                mlflow.sklearn.log_model(model, "model")
            
            # Log artifacts
            if artifacts:
                for name, path in artifacts.items():
                    mlflow.log_artifact(path, artifact_path=name)
            
            # Register model
            model_uri = f"runs:/{run.info.run_id}/model"
            mv = mlflow.register_model(model_uri, model_name)
            
            return {
                "version": mv.version,
                "run_id": run.info.run_id,
            }
    
    def load_model(self, model_name: str, version: Optional[str] = None) -> Any:
        """Load model from registry."""
        if version:
            model_uri = f"models:/{model_name}/{version}"
        else:
            model_uri = f"models:/{model_name}/latest"
        
        return mlflow.pyfunc.load_model(model_uri)
    
    def get_production_model(self, model_name: str) -> Any:
        """Get currently deployed production model."""
        model_uri = f"models:/{model_name}/Production"
        return mlflow.pyfunc.load_model(model_uri)
    
    def promote_model(self, model_name: str, version: str, stage: str = "Production"):
        """Promote model to specified stage."""
        self.client.transition_model_version_stage(
            name=model_name,
            version=version,
            stage=stage
        )
