"""Model training pipeline with hyperparameter tuning."""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Any, Dict, List, Optional, Tuple
import os

import numpy as np
import optuna
from optuna.trial import Trial
from sqlalchemy.ext.asyncio import AsyncSession

from app.preprocessing import (
    load_backtest_results,
    create_feature_dataframe,
    apply_feature_engineering,
    handle_missing_values,
    encode_categorical_features,
    normalize_features,
    create_train_test_split,
)
from app.config import get_settings
from app.model_registry import ModelRegistry
from app import model_io


@dataclass
class TrainingConfig:
    """Configuration for training pipeline."""
    model_type: str  # 'rl_agent', 'classifier', 'ensemble'
    data_filters: Dict[str, Any]
    epochs: int = 100
    batch_size: int = 32
    learning_rate: float = 0.001
    validation_split: float = 0.2
    test_split: float = 0.2
    n_trials: int = 50  # For hyperparameter search
    timeout: Optional[int] = 3600  # 1 hour timeout for optimization
    early_stopping_patience: int = 10
    random_state: int = 42


class TrainingPipeline:
    """Orchestrates model training with data loading and preprocessing."""
    
    def __init__(self, config: TrainingConfig):
        self.config = config
        self.scaler = None
        self.feature_names = None
    
    async def load_training_data(
        self,
        session: AsyncSession
    ) -> Tuple[np.ndarray, np.ndarray]:
        """Load and preprocess training data from database."""
        # Load backtest results with filters
        results = await load_backtest_results(session, self.config.data_filters)
        
        if not results:
            raise ValueError("No backtest results found with given filters")
        
        # Create feature dataframe
        df = create_feature_dataframe(results)
        df = apply_feature_engineering(df)
        df = handle_missing_values(df)
        df = encode_categorical_features(df)
        df, self.scaler = normalize_features(df)
        
        # Store feature names
        feature_cols = [col for col in df.columns if col not in ['id', 'strategy_id', 'run_date', 'recommendation']]
        self.feature_names = feature_cols
        
        # Extract features and targets
        X = df[feature_cols].values
        
        # Create binary target: 1 if composite_score > 0.7, else 0
        if 'composite_score' in df.columns:
            y = (df['composite_score'] > 0.7).astype(int).values
        else:
            raise ValueError("composite_score column not found in data")
        
        return X, y
    
    def prepare_features(
        self,
        X: np.ndarray,
        y: np.ndarray
    ) -> Tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
        """Create train/validation/test splits."""
        # First split: train+val vs test
        total_split = self.config.validation_split + self.config.test_split
        train_val_size = 1 - self.config.test_split
        
        X_train_val, X_test, y_train_val, y_test = create_train_test_split(
            X, y, test_size=self.config.test_split, temporal=True
        )
        
        # Second split: train vs val
        val_fraction = self.config.validation_split / train_val_size
        X_train, X_val, y_train, y_val = create_train_test_split(
            X_train_val, y_train_val, test_size=val_fraction, temporal=True
        )
        
        return X_train, X_val, X_test, y_train, y_val, y_test
    
    def hyperparameter_search(
        self,
        X_train: np.ndarray,
        y_train: np.ndarray,
        X_val: np.ndarray,
        y_val: np.ndarray,
    ) -> Dict[str, Any]:
        """Perform Bayesian optimization for hyperparameters."""
        
        def objective(trial: Trial) -> float:
            """Objective function to maximize validation Sharpe ratio."""
            if self.config.model_type == 'classifier':
                params = {
                    'learning_rate': trial.suggest_float('learning_rate', 1e-5, 1e-2, log=True),
                    'dropout_rate_1': trial.suggest_float('dropout_rate_1', 0.1, 0.5),
                    'dropout_rate_2': trial.suggest_float('dropout_rate_2', 0.1, 0.4),
                    'batch_size': trial.suggest_categorical('batch_size', [16, 32, 64, 128]),
                }
                
                from app.models.classifier import RaceOutcomeClassifier
                model = RaceOutcomeClassifier(
                    input_dim=X_train.shape[1],
                    learning_rate=params['learning_rate'],
                    dropout_rate_1=params['dropout_rate_1'],
                    dropout_rate_2=params['dropout_rate_2'],
                )
                
                model.train(
                    X_train, y_train,
                    validation_split=0.0,  # Use provided X_val
                    epochs=50,
                    batch_size=params['batch_size'],
                    early_stopping_patience=5,
                    verbose=0
                )
                
                metrics = model.evaluate(X_val, y_val)
                return metrics.get('auc', 0.0)
            
            elif self.config.model_type == 'ensemble':
                # For ensemble, optimize model selection and weights
                use_stacking = trial.suggest_categorical('use_stacking', [True, False])
                
                from app.models.ensemble import StrategyEnsemble
                ensemble = StrategyEnsemble(use_stacking=use_stacking)
                ensemble.train(X_train, y_train, feature_names=self.feature_names)
                
                from sklearn.metrics import roc_auc_score
                y_pred = ensemble.predict_proba(X_val)
                auc = roc_auc_score(y_val, y_pred)
                return auc
            
            else:  # rl_agent
                params = {
                    'learning_rate': trial.suggest_float('learning_rate', 1e-5, 1e-2, log=True),
                    'gamma': trial.suggest_float('gamma', 0.9, 0.999),
                    'epsilon_decay': trial.suggest_float('epsilon_decay', 0.99, 0.9999),
                    'batch_size': trial.suggest_categorical('batch_size', [32, 64, 128, 256]),
                }
                
                # Return dummy value for RL (would need full backtesting integration)
                return 0.5
        
        study = optuna.create_study(direction='maximize')
        study.optimize(
            objective,
            n_trials=self.config.n_trials,
            timeout=self.config.timeout,
            show_progress_bar=True
        )
        
        return study.best_params
    
    async def train_all_models(
        self,
        session: AsyncSession,
        use_hyperparameter_search: bool = False,
        X_train: Optional[np.ndarray] = None,
        X_val: Optional[np.ndarray] = None,
        X_test: Optional[np.ndarray] = None,
        y_train: Optional[np.ndarray] = None,
        y_val: Optional[np.ndarray] = None,
        y_test: Optional[np.ndarray] = None,
    ) -> Dict[str, Any]:
        """Train all specified models with MLflow registration."""
        settings = get_settings()
        os.makedirs(settings.model_cache_dir, exist_ok=True)

        # Load data if not provided
        if X_train is None or y_train is None or X_test is None or y_test is None:
            X, y = await self.load_training_data(session)
            X_train, X_val, X_test, y_train, y_val, y_test = self.prepare_features(X, y)

        # Hyperparameter search if requested
        best_params: Dict[str, Any] = {}
        if use_hyperparameter_search and X_val is not None and y_val is not None:
            best_params = self.hyperparameter_search(X_train, y_train, X_val, y_val)

        # Train model based on type
        model = None
        training_history: Dict[str, Any] = {}
        model_flavor = "sklearn"
        artifact_paths: Dict[str, str] = {}

        if self.config.model_type == 'classifier':
            from app.models.classifier import RaceOutcomeClassifier

            model = RaceOutcomeClassifier(
                input_dim=X_train.shape[1],
                **best_params
            )
            history = model.train(
                X_train, y_train,
                validation_split=0.0,
                epochs=self.config.epochs,
                batch_size=self.config.batch_size,
                early_stopping_patience=self.config.early_stopping_patience,
                verbose=1
            )
            training_history = history
            model_flavor = "tensorflow"

            # Persist artifacts
            model_path = f"{settings.model_cache_dir}/classifier_model"
            calibrator_path = f"{settings.model_cache_dir}/classifier_calibrator.joblib"
            model.save(model_path, calibrator_path)
            artifact_paths = {
                "model": model_path,
                "calibrator": calibrator_path,
            }

        elif self.config.model_type == 'ensemble':
            from app.models.ensemble import StrategyEnsemble

            use_stacking = best_params.get('use_stacking', True)
            model = StrategyEnsemble(use_stacking=use_stacking)
            model.train(X_train, y_train, feature_names=self.feature_names)
            model_flavor = "sklearn"

            # Persist artifacts
            model_path = f"{settings.model_cache_dir}/ensemble_model.joblib"
            model_io.save_sklearn_model(model.ensemble, model_path)
            artifact_paths = {"model": model_path}

        elif self.config.model_type == 'rl_agent':
            from app.models.rl_agent import train_rl_agent, DQNAgent

            # Convert data to backtest result format for RL
            backtest_results = self._prepare_rl_data(X_train, y_train)
            model = train_rl_agent(backtest_results, num_episodes=self.config.epochs)
            model_flavor = "pytorch"

            # Persist artifacts
            agent_path = f"{settings.model_cache_dir}/rl_agent.pt"
            model_io.save_dqn_agent(model, agent_path)
            artifact_paths = {"agent": agent_path}

        # Evaluate on test set
        test_metrics = self._evaluate_model(model, X_test, y_test)

        # Register model with MLflow
        registry = ModelRegistry(
            tracking_uri=settings.mlflow_tracking_uri,
            experiment_name=settings.mlflow_experiment_name
        )

        mlflow_model = model
        if self.config.model_type == 'ensemble':
            mlflow_model = model.ensemble
        elif self.config.model_type == 'classifier':
            mlflow_model = model.model
        elif self.config.model_type == 'rl_agent':
            mlflow_model = model.policy_net

        registration = registry.register_model(
            model=mlflow_model,
            model_name=self.config.model_type,
            model_type=model_flavor,
            metrics=test_metrics,
            params={**best_params, **self.config.data_filters},
            artifacts=artifact_paths
        )

        return {
            'model': model,
            'best_params': best_params,
            'training_history': training_history,
            'test_metrics': test_metrics,
            'feature_names': self.feature_names,
            'scaler': self.scaler,
            'model_version': registration.get('version'),
            'mlflow_run_id': registration.get('run_id'),
        }
    
    def _prepare_rl_data(self, X: np.ndarray, y: np.ndarray) -> List[Dict[str, Any]]:
        """Convert features to RL-compatible format."""
        results = []
        for i in range(len(X)):
            result = {
                'composite_score': float(y[i]),
                'total_return': X[i][0] if X.shape[1] > 0 else 0.0,
                'sharpe_ratio': X[i][1] if X.shape[1] > 1 else 0.0,
                'win_rate': X[i][2] if X.shape[1] > 2 else 0.5,
                'profit_factor': X[i][3] if X.shape[1] > 3 else 1.0,
                'full_results': {
                    'market': {'avg_odds': 2.0, 'avg_volume': 1000},
                    'risk_profile': {'var_95': 0.05},
                    'equity_curve': [{'value': 1000}, {'value': 1050}],
                },
            }
            results.append(result)
        return results
    
    def _evaluate_model(
        self,
        model: Any,
        X_test: np.ndarray,
        y_test: np.ndarray
    ) -> Dict[str, float]:
        """Evaluate model on test set."""
        from sklearn.metrics import accuracy_score, precision_score, recall_score, f1_score, roc_auc_score
        
        if hasattr(model, 'predict_probabilities'):
            y_pred_proba = model.predict_probabilities(X_test, calibrated=True)
            y_pred = (y_pred_proba > 0.5).astype(int)
        elif hasattr(model, 'predict_proba'):
            y_pred_proba = model.predict_proba(X_test)
            y_pred = (y_pred_proba > 0.5).astype(int)
        else:
            y_pred = model.predict(X_test) if hasattr(model, 'predict') else np.zeros_like(y_test)
            y_pred_proba = y_pred
        
        metrics = {
            'accuracy': float(accuracy_score(y_test, y_pred)),
            'precision': float(precision_score(y_test, y_pred, zero_division=0)),
            'recall': float(recall_score(y_test, y_pred, zero_division=0)),
            'f1_score': float(f1_score(y_test, y_pred, zero_division=0)),
        }
        
        try:
            metrics['roc_auc'] = float(roc_auc_score(y_test, y_pred_proba))
        except:
            metrics['roc_auc'] = 0.0
        
        return metrics
    
    async def incremental_training(
        self,
        session: AsyncSession,
        existing_model: Any,
        new_data_filters: Dict[str, Any]
    ) -> Any:
        """Update model with new data without full retraining."""
        # Load new data
        new_results = await load_backtest_results(session, new_data_filters)
        
        if not new_results:
            return existing_model
        
        # Process new data
        df = create_feature_dataframe(new_results)
        df = apply_feature_engineering(df)
        df = handle_missing_values(df)
        df = encode_categorical_features(df)
        df, _ = normalize_features(df, scaler=self.scaler)
        
        X_new = df[self.feature_names].values
        y_new = (df['composite_score'] > 0.7).astype(int).values
        
        # Incremental update (model-specific)
        if hasattr(existing_model, 'train'):
            # For classifier, continue training
            existing_model.train(
                X_new, y_new,
                validation_split=0.1,
                epochs=10,  # Fewer epochs for incremental
                verbose=0
            )
        elif hasattr(existing_model, 'partial_fit'):
            # For sklearn models with partial_fit
            existing_model.partial_fit(X_new, y_new)
        
        return existing_model


def create_train_test_split(
    X: np.ndarray,
    y: np.ndarray,
    test_size: float = 0.2,
    temporal: bool = True
) -> Tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    """Create train/test split maintaining temporal order."""
    if temporal:
        split_idx = int(len(X) * (1 - test_size))
        X_train, X_test = X[:split_idx], X[split_idx:]
        y_train, y_test = y[:split_idx], y[split_idx:]
    else:
        from sklearn.model_selection import train_test_split as sklearn_split
        X_train, X_test, y_train, y_test = sklearn_split(
            X, y, test_size=test_size, random_state=42, stratify=y
        )
    
    return X_train, X_test, y_train, y_test
