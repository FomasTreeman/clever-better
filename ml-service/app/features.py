from __future__ import annotations

from typing import Any, Dict, List

import numpy as np


def extract_race_features(full_results: Dict[str, Any]) -> Dict[str, Any]:
    race = full_results.get("race", {})
    return {
        "track_id": race.get("track_id"),
        "distance": race.get("distance"),
        "grade": race.get("grade"),
        "time_of_day": race.get("time_of_day"),
        "field_size": race.get("field_size"),
    }


def extract_runner_features(full_results: Dict[str, Any]) -> Dict[str, Any]:
    runner = full_results.get("runner", {})
    return {
        "trap_number": runner.get("trap_number"),
        "recent_form": runner.get("recent_form"),
        "win_rate_track": runner.get("win_rate_track"),
        "days_since_race": runner.get("days_since_race"),
    }


def extract_market_features(full_results: Dict[str, Any]) -> Dict[str, Any]:
    market = full_results.get("market", {})
    return {
        "avg_odds": market.get("avg_odds"),
        "avg_volume": market.get("avg_volume"),
        "odds_drift": market.get("odds_drift"),
    }


def create_interaction_features(df):
    if "avg_odds" in df.columns and "recent_form" in df.columns:
        df["odds_vs_form"] = df["avg_odds"] * df["recent_form"]
    if "trap_number" in df.columns and "grade" in df.columns:
        df["trap_grade_interaction"] = df["trap_number"].astype(float)
    return df


def calculate_risk_metrics(full_results: Dict[str, Any]) -> Dict[str, Any]:
    risk = full_results.get("risk_profile", {})
    return {
        "var_95": risk.get("var_95"),
        "var_99": risk.get("var_99"),
        "tail_risk": risk.get("tail_risk"),
    }


def calculate_consistency_metrics(equity_curve: List[Dict[str, Any]]) -> Dict[str, Any]:
    if not equity_curve:
        return {"consistency_score": 0.0}
    values = np.array([point.get("value", 0) for point in equity_curve])
    if values.size < 2:
        return {"consistency_score": 0.0}
    returns = np.diff(values) / np.maximum(values[:-1], 1e-6)
    positive = np.sum(returns > 0)
    return {"consistency_score": float(positive / returns.size)}


def aggregate_strategy_features(strategy_results: List[Dict[str, Any]]) -> Dict[str, Any]:
    if not strategy_results:
        return {}
    composite_scores = [r.get("composite_score", 0) for r in strategy_results]
    return {
        "avg_composite_score": float(np.mean(composite_scores)),
        "max_composite_score": float(np.max(composite_scores)),
        "min_composite_score": float(np.min(composite_scores)),
    }


def create_feature_vector(backtest_result: Dict[str, Any]) -> Dict[str, Any]:
    full_results = backtest_result.get("full_results", {})
    features = {}
    features.update(extract_race_features(full_results))
    features.update(extract_runner_features(full_results))
    features.update(extract_market_features(full_results))
    features.update(calculate_risk_metrics(full_results))
    features.update(calculate_consistency_metrics(full_results.get("equity_curve", [])))
    return features
