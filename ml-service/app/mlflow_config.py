"""MLflow configuration and initialization."""
from __future__ import annotations

import mlflow
from mlflow.tracking import MlflowClient


def init_mlflow(tracking_uri: str, experiment_name: str):
    """Initialize MLflow tracking."""
    mlflow.set_tracking_uri(tracking_uri)
    
    # Create experiment if it doesn't exist
    client = MlflowClient(tracking_uri)
    try:
        experiment = client.get_experiment_by_name(experiment_name)
        if experiment is None:
            mlflow.create_experiment(experiment_name)
    except Exception:
        mlflow.create_experiment(experiment_name)
    
    mlflow.set_experiment(experiment_name)
    
    return client


def log_model_metrics(metrics: dict, step: int = None):
    """Log metrics to MLflow."""
    for key, value in metrics.items():
        mlflow.log_metric(key, value, step=step)


def log_model_params(params: dict):
    """Log parameters to MLflow."""
    mlflow.log_params(params)


def log_artifact(local_path: str, artifact_path: str = None):
    """Log artifact to MLflow."""
    mlflow.log_artifact(local_path, artifact_path=artifact_path)
