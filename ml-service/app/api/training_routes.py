"""Training API endpoints."""
from __future__ import annotations

from datetime import datetime
from typing import Dict, Any, Optional
from fastapi import APIRouter, BackgroundTasks, HTTPException
from pydantic import BaseModel

from app.training import TrainingPipeline, TrainingConfig
from app.database import get_db


router = APIRouter(prefix="/models", tags=["training"])

# Track training jobs
training_jobs: Dict[str, Dict[str, Any]] = {}


class TrainRequest(BaseModel):
    model_type: str
    config: Optional[Dict[str, Any]] = None
    hyperparameter_search: bool = False
    n_trials: int = 50


class TrainResponse(BaseModel):
    job_id: str
    status: str
    message: str


async def run_training_job(job_id: str, request: TrainRequest):
    """Background training task."""
    try:
        training_jobs[job_id]['status'] = 'running'
        training_jobs[job_id]['started_at'] = datetime.utcnow().isoformat()
        
        # Create pipeline
        request_config = request.config or {}
        data_filters = request_config.get("data_filters", {})

        config = TrainingConfig(
            model_type=request.model_type,
            data_filters=data_filters,
            epochs=request_config.get("epochs", 100),
            batch_size=request_config.get("batch_size", 32),
            learning_rate=request_config.get("learning_rate", 0.001),
            validation_split=request_config.get("validation_split", 0.2),
            test_split=request_config.get("test_split", 0.2),
            n_trials=request.n_trials,
            early_stopping_patience=request_config.get("early_stopping_patience", 10),
        )
        pipeline = TrainingPipeline(config)

        async with get_db() as session:
            # Load and preprocess data using existing pipeline methods
            X, y = await pipeline.load_training_data(session)
            X_train, X_val, X_test, y_train, y_val, y_test = pipeline.prepare_features(X, y)

            # Train and register model
            result = await pipeline.train_all_models(
                session=session,
                use_hyperparameter_search=request.hyperparameter_search,
                X_train=X_train,
                X_val=X_val,
                X_test=X_test,
                y_train=y_train,
                y_val=y_val,
                y_test=y_test,
            )

        training_jobs[job_id]['status'] = 'completed'
        training_jobs[job_id]['metrics'] = result.get('test_metrics')
        training_jobs[job_id]['best_params'] = result.get('best_params')
        training_jobs[job_id]['model_version'] = result.get('model_version')
        training_jobs[job_id]['mlflow_run_id'] = result.get('mlflow_run_id')
        training_jobs[job_id]['completed_at'] = datetime.utcnow().isoformat()
        
    except Exception as e:
        training_jobs[job_id]['status'] = 'failed'
        training_jobs[job_id]['error'] = str(e)
        training_jobs[job_id]['failed_at'] = datetime.utcnow().isoformat()


@router.post("/train", response_model=TrainResponse)
async def train_model(request: TrainRequest, background_tasks: BackgroundTasks):
    """Start model training job."""
    job_id = f"train_{request.model_type}_{datetime.utcnow().timestamp()}"
    
    training_jobs[job_id] = {
        'status': 'pending',
        'model_type': request.model_type,
        'created_at': datetime.utcnow().isoformat()
    }
    
    background_tasks.add_task(run_training_job, job_id, request)
    
    return TrainResponse(
        job_id=job_id,
        status='pending',
        message=f'Training job {job_id} started'
    )


@router.get("/training/{job_id}")
async def get_training_status(job_id: str):
    """Get training job status."""
    if job_id not in training_jobs:
        raise HTTPException(status_code=404, detail="Job not found")
    
    return training_jobs[job_id]


@router.get("/training")
async def list_training_jobs():
    """List all training jobs."""
    return list(training_jobs.values())
