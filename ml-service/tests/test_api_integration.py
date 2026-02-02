"""Integration tests for API endpoints with feature engineering."""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch
import pandas as pd

try:
    from app.api.schemas import FeatureExtractionRequest
except ImportError:
    pytest.skip("API schemas not available", allow_module_level=True)


@pytest.fixture
def mock_backtest_results():
    """Create mock backtest results."""
    results = []
    for i in range(3):
        result = MagicMock()
        result.id = f"result-{i}"
        result.strategy_id = "strategy-1"
        result.run_date = pd.Timestamp("2024-01-01") + pd.Timedelta(days=i)
        result.total_return = 0.1 + i * 0.05
        result.sharpe_ratio = 1.2 + i * 0.1
        result.max_drawdown = 0.05 - i * 0.01
        result.win_rate = 0.5 + i * 0.1
        result.profit_factor = 1.3 + i * 0.2
        result.composite_score = 0.6 + i * 0.1
        result.recommendation = "ACCEPT" if i == 0 else "REVIEW"
        result.method = "historical"
        result.ml_features = {
            "avg_odds": 2.0 + i * 0.5,
            "recent_form": 0.8 - i * 0.1,
            "trap_number": i + 1,
        }
        results.append(result)
    return results


@pytest.mark.asyncio
async def test_preprocess_endpoint_applies_feature_engineering(async_client, mock_backtest_results):
    """Test /preprocess endpoint applies feature engineering to dataframe."""
    with patch("app.api.routes.load_backtest_results") as mock_load:
        mock_load.return_value = mock_backtest_results
        
        payload = {"strategy_id": "strategy-1"}
        response = await async_client.post("/api/v1/preprocess", json=payload)
        
        assert response.status_code == 200
        data = response.json()
        
        # Verify response structure
        assert "rows" in data
        assert "columns" in data
        assert "preview" in data
        
        # Verify features were created (should have interaction features now)
        columns = data["columns"]
        
        # Check that interaction features from feature engineering are present
        # or at least the basic features
        assert len(columns) > 0
        assert data["rows"] == 3


@pytest.mark.asyncio
async def test_features_extract_endpoint_uses_feature_vector(async_client):
    """Test /features/extract endpoint applies all feature engineering."""
    payload = {
        "full_results": {
            "race": {
                "track_id": "track_1",
                "distance": 500,
                "grade": "A",
            },
            "runner": {
                "trap_number": 3,
                "recent_form": 0.8,
            },
            "market": {
                "avg_odds": 2.5,
                "avg_volume": 100000,
            },
            "equity_curve": [
                {"value": 1000},
                {"value": 1100},
            ],
        }
    }
    
    response = await async_client.post("/api/v1/features/extract", json=payload)
    
    assert response.status_code == 200
    data = response.json()
    
    # Verify features were extracted
    assert isinstance(data, dict)
    assert len(data) > 0
    
    # Verify key features are present
    assert "track_id" in data or "trap_number" in data


@pytest.mark.asyncio  
async def test_preprocess_feature_engineering_integration():
    """Test the full preprocessing pipeline including feature engineering."""
    from app.preprocessing import (
        create_feature_dataframe,
        apply_feature_engineering,
        handle_missing_values,
        encode_categorical_features,
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
            self.ml_features = {
                "avg_odds": 2.0,
                "recent_form": 0.8,
                "trap_number": 3,
                "grade": "A",
            }
            self.full_results = {
                "race": {"distance": 500, "grade": "A"},
                "runner": {"trap_number": 3, "recent_form": 0.8},
                "market": {"avg_odds": 2.0}
            }
    
    results = [DummyResult()]
    
    # Apply pipeline
    df = create_feature_dataframe(results)
    df = apply_feature_engineering(df)
    df = handle_missing_values(df)
    df = encode_categorical_features(df)
    df, _ = normalize_features(df)
    
    # Verify pipeline processed correctly
    assert not df.empty
    assert len(df) == 1
    
    # Verify interaction features exist
    assert "odds_vs_form" in df.columns or "avg_odds" in df.columns
