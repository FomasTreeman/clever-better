"""Dashboard routes for aggregated monitoring data."""

from fastapi import APIRouter

router = APIRouter(prefix="/dashboard", tags=["dashboard"])


@router.get("/overview")
async def overview():
    """Get summary metrics for main dashboard."""
    return {
        "bankroll": {
            "current": 10500,
            "initial": 10000,
            "change_percent": 5.0,
        },
        "pnl": {
            "daily": 500,
            "weekly": 1200,
            "monthly": 3400,
        },
        "active_strategies": 8,
        "circuit_breaker": {
            "status": "NORMAL",
            "trips_today": 0,
        },
        "total_bets_placed": 1250,
        "win_rate": 0.563,
    }


@router.get("/strategy-performance")
async def strategy_performance():
    """Get strategy performance time series data for dashboard."""
    return {
        "timestamp": "2024-02-03T12:00:00Z",
        "strategies": [
            {
                "strategy_id": "strategy_001",
                "name": "SimpleValueStrategy",
                "pnl": 450,
                "active_bets": 12,
                "signals_last_hour": 5,
                "win_rate": 0.58,
            }
        ],
    }


@router.get("/ml-health")
async def ml_health():
    """Get ML service health and metrics."""
    return {
        "ml_service": {
            "status": "HEALTHY",
            "uptime_seconds": 86400,
            "last_health_check": "2024-02-03T12:00:00Z",
        },
        "prediction_metrics": {
            "cache_hit_ratio": 0.78,
            "p50_latency_ms": 45,
            "p95_latency_ms": 230,
            "p99_latency_ms": 450,
            "error_rate": 0.002,
        },
        "training_jobs": {
            "active": 1,
            "completed_today": 2,
            "failed": 0,
        },
    }


@router.get("/recent-decisions")
async def recent_decisions():
    """Get recent strategy decisions with context."""
    return {
        "decisions": [
            {
                "timestamp": "2024-02-03T12:00:00Z",
                "strategy_id": "strategy_001",
                "decision": "PLACE_BET",
                "confidence": 0.87,
                "edge_value": 0.045,
                "stake": 100,
                "outcome": "PENDING",
            }
        ]
    }
