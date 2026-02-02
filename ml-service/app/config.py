from __future__ import annotations

from functools import lru_cache
from typing import List

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables or .env."""

    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8")

    database_url: str = Field(..., alias="DATABASE_URL")
    environment: str = Field("development", alias="ENVIRONMENT")
    log_level: str = Field("INFO", alias="LOG_LEVEL")
    grpc_port: int = Field(50051, alias="GRPC_PORT")
    api_host: str = Field("0.0.0.0", alias="API_HOST")
    api_port: int = Field(8000, alias="API_PORT")
    cors_origins: List[str] = Field(default_factory=lambda: ["*"])

    db_pool_min_size: int = Field(1, alias="DB_POOL_MIN_SIZE")
    db_pool_max_size: int = Field(10, alias="DB_POOL_MAX_SIZE")

    # ML-specific settings
    mlflow_tracking_uri: str = Field("http://localhost:5000", alias="MLFLOW_TRACKING_URI")
    mlflow_experiment_name: str = Field("clever-better", alias="MLFLOW_EXPERIMENT_NAME")
    model_cache_dir: str = Field("/tmp/model_cache", alias="MODEL_CACHE_DIR")
    max_training_workers: int = Field(2, alias="MAX_TRAINING_WORKERS")
    prediction_timeout_seconds: int = Field(30, alias="PREDICTION_TIMEOUT_SECONDS")
    
    # Monitoring settings
    enable_metrics: bool = Field(True, alias="ENABLE_METRICS")
    enable_structured_logging: bool = Field(True, alias="ENABLE_STRUCTURED_LOGGING")

    @field_validator("cors_origins", mode="before")
    @classmethod
    def parse_cors_origins(cls, value):
        if isinstance(value, str):
            return [origin.strip() for origin in value.split(",") if origin.strip()]
        return value


@lru_cache
def get_settings() -> Settings:
    return Settings()
