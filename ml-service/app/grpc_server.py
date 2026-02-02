from __future__ import annotations

import asyncio
import math
from typing import Any

import grpc
from sqlalchemy import select

from app.config import get_settings
from app.database import database_health_check, engine, get_session
from app.features import aggregate_strategy_features, create_feature_vector
from app.models.db_models import BacktestResult
from app.utils.logging import configure_logging, get_logger

try:
    from app.generated import ml_service_pb2, ml_service_pb2_grpc
except ImportError as exc:  # pragma: no cover
    raise RuntimeError(
        "gRPC modules not generated. Run 'make proto-gen' to generate gRPC code."
    ) from exc

settings = get_settings()
configure_logging(settings.log_level)
logger = get_logger(__name__)


class MLServiceServicer(ml_service_pb2_grpc.MLServiceServicer):
    def __init__(self, engine_instance):
        self.engine = engine_instance

    async def GetPrediction(self, request: ml_service_pb2.PredictionRequest, context: grpc.aio.ServicerContext):
        """Compute predicted probability using mock ML (sigmoid on avg features)."""
        try:
            if not request.race_id or not request.strategy_id:
                await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "race_id and strategy_id required")
                return
            
            # Mock ML: sigmoid on average feature value
            if request.features:
                avg_feat = sum(request.features) / len(request.features)
            else:
                avg_feat = 0.0
            
            # Sigmoid activation for probability
            predicted_probability = 1.0 / (1.0 + math.exp(-avg_feat))
            confidence = min(1.0, len(request.features) / 10.0)
            
            logger.info(
                "grpc_prediction",
                race_id=request.race_id,
                strategy_id=request.strategy_id,
                feature_count=len(request.features),
                probability=predicted_probability,
            )
            
            return ml_service_pb2.PredictionResponse(
                race_id=request.race_id,
                predicted_probability=predicted_probability,
                confidence=confidence,
            )
        except Exception as exc:
            logger.error("grpc_prediction_error", error=str(exc))
            await context.abort(grpc.StatusCode.INTERNAL, f"Prediction failed: {exc}")

    async def EvaluateStrategy(self, request: ml_service_pb2.StrategyRequest, context: grpc.aio.ServicerContext):
        """Aggregate backtest results and compute composite score/recommendation."""
        try:
            if not request.strategy_id:
                await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "strategy_id required")
                return
            
            async with get_session(self.engine) as session:
                query = select(BacktestResult).where(BacktestResult.strategy_id == request.strategy_id)
                result = await session.execute(query)
                records = list(result.scalars().all())
                
                if not records:
                    context.set_code(grpc.StatusCode.NOT_FOUND)
                    context.set_details(f"No backtest results for strategy {request.strategy_id}")
                    return ml_service_pb2.StrategyResponse(
                        strategy_id=request.strategy_id,
                        composite_score=0.0,
                        recommendation="NOT_FOUND",
                    )
                
                # Aggregate using features.py
                agg = aggregate_strategy_features([
                    {"composite_score": r.composite_score}
                    for r in records
                ])
                
                avg_score = agg.get("avg_composite_score", 0.0)
                recommendation = "APPROVED" if avg_score > 0.7 else "NEEDS_REVIEW"
                
                logger.info(
                    "grpc_strategy_evaluated",
                    strategy_id=request.strategy_id,
                    result_count=len(records),
                    avg_score=avg_score,
                    recommendation=recommendation,
                )
                
                return ml_service_pb2.StrategyResponse(
                    strategy_id=request.strategy_id,
                    composite_score=avg_score,
                    recommendation=recommendation,
                )
        except Exception as exc:
            logger.error("grpc_evaluate_error", error=str(exc))
            await context.abort(grpc.StatusCode.INTERNAL, f"Evaluation failed: {exc}")

    async def GetFeatures(self, request: ml_service_pb2.FeatureRequest, context: grpc.aio.ServicerContext):
        """Load backtest result and extract engineered features."""
        try:
            if not request.backtest_result_id:
                await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "backtest_result_id required")
                return
            
            async with get_session(self.engine) as session:
                result = await session.get(BacktestResult, request.backtest_result_id)
                
                if not result:
                    context.set_code(grpc.StatusCode.NOT_FOUND)
                    context.set_details(f"Backtest result {request.backtest_result_id} not found")
                    return ml_service_pb2.FeatureResponse(features={})
                
                # Extract features using features.py
                full_results_dict = dict(result.full_results) if result.full_results else {}
                features = create_feature_vector({"full_results": full_results_dict})
                
                # Convert to proto map (only numeric values)
                feat_map = {}
                for k, v in features.items():
                    if isinstance(v, (int, float)):
                        feat_map[k] = float(v)
                    elif v is None:
                        feat_map[k] = 0.0
                
                logger.info(
                    "grpc_features_extracted",
                    backtest_result_id=request.backtest_result_id,
                    feature_count=len(feat_map),
                )
                
                return ml_service_pb2.FeatureResponse(features=feat_map)
        except Exception as exc:
            logger.error("grpc_get_features_error", error=str(exc))
            await context.abort(grpc.StatusCode.INTERNAL, f"Feature extraction failed: {exc}")

    async def HealthCheck(self, request: ml_service_pb2.Empty, context: grpc.aio.ServicerContext):
        db_ok = await database_health_check()
        status = "ok" if db_ok else "degraded"
        return ml_service_pb2.HealthStatus(status=status)


class LoggingInterceptor(grpc.aio.ServerInterceptor):
    async def intercept_service(self, continuation, handler_call_details):
        handler = await continuation(handler_call_details)
        if handler is None:
            return None

        if handler.unary_unary:
            async def unary_unary(request, context):
                logger.info("grpc_request", method=handler_call_details.method)
                try:
                    return await handler.unary_unary(request, context)
                except Exception as exc:
                    logger.error("grpc_error", method=handler_call_details.method, error=str(exc))
                    raise

            return grpc.aio.unary_unary_rpc_method_handler(
                unary_unary,
                request_deserializer=handler.request_deserializer,
                response_serializer=handler.response_serializer,
            )

        return handler


async def serve() -> None:
    server = grpc.aio.server(interceptors=[LoggingInterceptor()])
    ml_service_pb2_grpc.add_MLServiceServicer_to_server(MLServiceServicer(engine), server)
    server.add_insecure_port(f"0.0.0.0:{settings.grpc_port}")
    logger.info("grpc_server_starting", port=settings.grpc_port)
    await server.start()
    await server.wait_for_termination()


if __name__ == "__main__":
    asyncio.run(serve())
