"""Integration tests for gRPC servicer with real database and feature engineering."""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.db_models import BacktestResult
from app.grpc_server import MLServiceServicer

try:
    from app.generated import ml_service_pb2
except ImportError:
    pytest.skip("gRPC modules not generated", allow_module_level=True)


@pytest.fixture
def mock_backtest_result():
    """Create a mock BacktestResult with full_results JSONB."""
    result = MagicMock(spec=BacktestResult)
    result.id = "test-result-id"
    result.strategy_id = "test-strategy-id"
    result.composite_score = 0.75
    result.full_results = {
        "race": {"track_id": "track_1", "distance": 500},
        "runner": {"trap_number": 3, "recent_form": 0.8},
        "market": {"avg_odds": 2.5},
        "risk_profile": {"var_95": 0.15},
        "equity_curve": [{"value": 1000}, {"value": 1050}],
    }
    return result


@pytest.mark.asyncio
async def test_grpc_get_features_loads_database(mock_backtest_result):
    """Test GetFeatures loads backtest result and extracts features."""
    mock_engine = AsyncMock()
    servicer = MLServiceServicer(mock_engine)
    
    mock_session = AsyncMock(spec=AsyncSession)
    mock_session.get = AsyncMock(return_value=mock_backtest_result)
    mock_session.__aenter__ = AsyncMock(return_value=mock_session)
    mock_session.__aexit__ = AsyncMock(return_value=None)
    
    with patch("app.grpc_server.get_session") as mock_get_session:
        mock_get_session.return_value = mock_session
        
        request = ml_service_pb2.FeatureRequest(backtest_result_id="test-result-id")
        context = AsyncMock()
        
        response = await servicer.GetFeatures(request, context)
        
        mock_session.get.assert_called_once_with(BacktestResult, "test-result-id")
        assert response.features is not None
        assert isinstance(response.features, dict)


@pytest.mark.asyncio
async def test_grpc_get_features_handles_missing_result():
    """Test GetFeatures returns NOT_FOUND for missing backtest result."""
    mock_engine = AsyncMock()
    servicer = MLServiceServicer(mock_engine)
    
    mock_session = AsyncMock(spec=AsyncSession)
    mock_session.get = AsyncMock(return_value=None)
    mock_session.__aenter__ = AsyncMock(return_value=mock_session)
    mock_session.__aexit__ = AsyncMock(return_value=None)
    
    with patch("app.grpc_server.get_session") as mock_get_session:
        mock_get_session.return_value = mock_session
        
        request = ml_service_pb2.FeatureRequest(backtest_result_id="nonexistent-id")
        context = AsyncMock()
        context.set_code = MagicMock()
        context.set_details = MagicMock()
        
        response = await servicer.GetFeatures(request, context)
        
        context.set_code.assert_called()
        assert response.features == {}


@pytest.mark.asyncio
async def test_grpc_evaluate_strategy_loads_results():
    """Test EvaluateStrategy loads backtest results and aggregates composite score."""
    mock_engine = AsyncMock()
    servicer = MLServiceServicer(mock_engine)
    
    mock_result1 = MagicMock(spec=BacktestResult)
    mock_result1.composite_score = 0.75
    
    mock_result2 = MagicMock(spec=BacktestResult)
    mock_result2.composite_score = 0.85
    
    mock_scalars = MagicMock()
    mock_scalars.all.return_value = [mock_result1, mock_result2]
    
    mock_exec_result = MagicMock()
    mock_exec_result.scalars.return_value = mock_scalars
    
    mock_session = AsyncMock(spec=AsyncSession)
    mock_session.execute = AsyncMock(return_value=mock_exec_result)
    mock_session.__aenter__ = AsyncMock(return_value=mock_session)
    mock_session.__aexit__ = AsyncMock(return_value=None)
    
    with patch("app.grpc_server.get_session") as mock_get_session:
        mock_get_session.return_value = mock_session
        
        request = ml_service_pb2.StrategyRequest(strategy_id="test-strategy-id")
        context = AsyncMock()
        
        response = await servicer.EvaluateStrategy(request, context)
        
        assert response.strategy_id == "test-strategy-id"
        assert response.composite_score == 0.8
        assert response.recommendation in ["APPROVED", "NEEDS_REVIEW"]


@pytest.mark.asyncio
async def test_grpc_get_prediction_uses_features():
    """Test GetPrediction uses feature data (sigmoid on avg)."""
    mock_engine = AsyncMock()
    servicer = MLServiceServicer(mock_engine)
    
    request = ml_service_pb2.PredictionRequest(
        race_id="race-123",
        strategy_id="strategy-456",
        features=[0.5, 0.6, 0.7],
    )
    context = AsyncMock()
    
    response = await servicer.GetPrediction(request, context)
    
    assert response.race_id == "race-123"
    assert 0.0 <= response.predicted_probability <= 1.0
    assert response.predicted_probability != 0.5
    assert response.confidence == 0.3
