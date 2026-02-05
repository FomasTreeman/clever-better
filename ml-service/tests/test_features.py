import pytest

from app.features import (
    calculate_consistency_metrics,
    calculate_risk_metrics,
    create_feature_vector,
)


def test_calculate_risk_metrics():
    full_results = {"risk_profile": {"var_95": 0.1, "var_99": 0.2, "tail_risk": 0.3}}
    metrics = calculate_risk_metrics(full_results)
    assert metrics["var_95"] == pytest.approx(0.1)


def test_calculate_consistency_metrics():
    equity_curve = [{"value": 100}, {"value": 105}, {"value": 102}]
    metrics = calculate_consistency_metrics(equity_curve)
    assert "consistency_score" in metrics


def test_create_feature_vector():
    backtest_result = {"full_results": {"equity_curve": []}}
    vector = create_feature_vector(backtest_result)
    assert isinstance(vector, dict)
