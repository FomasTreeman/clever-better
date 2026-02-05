from __future__ import annotations

import time

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, Response
from sqlalchemy.exc import SQLAlchemyError
from fastapi.middleware.cors import CORSMiddleware

from app.api.routes import router
from app.config import get_settings
from app.database import database_health_check
from app.utils.logging import configure_logging, get_logger
from app.api.training_routes import router as training_router
from app.api.prediction_routes import router as prediction_router
from app.api.visualization_routes import router as visualization_router
from app.api.dashboard_routes import router as dashboard_router
from app.monitoring import ml_logger
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST
import mlflow

API_PREFIX = "/api/v1"

settings = get_settings()
configure_logging(settings.log_level)
logger = get_logger(__name__)

app = FastAPI(
    title="Clever Better ML Service",
    version="1.0.0",
    description="ML service for strategy evaluation and feature extraction",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(router)
app.include_router(training_router, prefix=API_PREFIX)
app.include_router(prediction_router, prefix=API_PREFIX)
app.include_router(visualization_router, prefix=API_PREFIX)
app.include_router(dashboard_router, prefix=API_PREFIX)


@app.get("/health")
async def health() -> dict:
    db_ok = await database_health_check()
    return {"status": "ok", "database": db_ok}


@app.get("/metrics")
async def metrics() -> Response:
    return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)


@app.middleware("http")
async def log_requests(request: Request, call_next):
    start = time.time()
    response = await call_next(request)
    duration_ms = (time.time() - start) * 1000
    logger.info(
        "request_completed",
        method=request.method,
        path=request.url.path,
        status_code=response.status_code,
        duration_ms=round(duration_ms, 2),
    )
    return response


@app.on_event("startup")
async def on_startup() -> None:
    logger.info("ml_service_starting")
    
    # Initialize MLflow
    mlflow.set_tracking_uri(settings.mlflow_tracking_uri)
    mlflow.set_experiment(settings.mlflow_experiment_name)
    ml_logger.logger.info(f"MLflow initialized: {settings.mlflow_tracking_uri}")


@app.on_event("shutdown")
async def on_shutdown() -> None:
    logger.info("ml_service_shutting_down")


@app.exception_handler(SQLAlchemyError)
async def sqlalchemy_exception_handler(request: Request, exc: SQLAlchemyError):
    logger.error("database_error", error=str(exc))
    return JSONResponse(status_code=500, content={"detail": "Database error"})
