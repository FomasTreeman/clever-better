"""Prometheus metrics for ML service."""
from __future__ import annotations

from prometheus_client import Counter, Histogram, Gauge


# Prediction metrics
prediction_counter = Counter(
    'ml_predictions_total',
    'Total number of predictions made',
    ['model_name', 'status']
)

prediction_latency = Histogram(
    'ml_prediction_latency_seconds',
    'Prediction latency in seconds',
    ['model_name']
)

prediction_confidence = Gauge(
    'ml_prediction_confidence',
    'Confidence of last prediction',
    ['model_name']
)

# Training metrics
training_counter = Counter(
    'ml_training_jobs_total',
    'Total number of training jobs',
    ['model_type', 'status']
)

training_duration = Histogram(
    'ml_training_duration_seconds',
    'Training duration in seconds',
    ['model_type']
)

model_accuracy = Gauge(
    'ml_model_accuracy',
    'Model accuracy metric',
    ['model_name', 'dataset']
)

# Model registry metrics
registered_models = Gauge(
    'ml_registered_models_total',
    'Total number of registered models'
)

model_versions = Gauge(
    'ml_model_versions',
    'Number of versions per model',
    ['model_name']
)
