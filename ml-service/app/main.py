from __future__ import annotations

import time

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from sqlalchemy.exc import SQLAlchemyError
from fastapi.middleware.cors import CORSMiddleware

from app.api.routes import router
from app.config import get_settings
from app.database import database_health_check
from app.utils.logging import configure_logging, get_logger

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


@app.get("/health")
async def health() -> dict:
    db_ok = await database_health_check()
    return {"status": "ok", "database": db_ok}


@app.get("/metrics")
async def metrics() -> dict:
    return {"status": "ok"}


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


@app.on_event("shutdown")
async def on_shutdown() -> None:
    logger.info("ml_service_shutting_down")


@app.exception_handler(SQLAlchemyError)
async def sqlalchemy_exception_handler(request: Request, exc: SQLAlchemyError):
    logger.error("database_error", error=str(exc))
    return JSONResponse(status_code=500, content={"detail": "Database error"})
