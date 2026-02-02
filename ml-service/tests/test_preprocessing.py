import pandas as pd

from app.preprocessing import (
    create_feature_dataframe,
    encode_categorical_features,
    handle_missing_values,
    normalize_features,
)


class DummyResult:
    def __init__(self):
        self.id = "1"
        self.strategy_id = "s1"
        self.run_date = pd.Timestamp("2024-01-01")
        self.total_return = 0.1
        self.sharpe_ratio = 1.2
        self.max_drawdown = 0.05
        self.win_rate = 0.5
        self.profit_factor = 1.3
        self.composite_score = 0.7
        self.recommendation = "ACCEPT"
        self.method = "historical"
        self.ml_features = {"feature_a": 1.0}


def test_create_feature_dataframe():
    df = create_feature_dataframe([DummyResult()])
    assert not df.empty
    assert "feature_a" in df.columns


def test_handle_missing_values():
    df = pd.DataFrame({"a": [1, None]})
    df = handle_missing_values(df)
    assert df.isna().sum().sum() == 0


def test_encode_categorical_features():
    df = pd.DataFrame({"recommendation": ["ACCEPT"], "method": ["historical"]})
    df = encode_categorical_features(df)
    assert any(col.startswith("recommendation") for col in df.columns)


def test_normalize_features():
    df = pd.DataFrame({"x": [1.0, 2.0, 3.0]})
    df, scaler = normalize_features(df)
    assert scaler is not None
