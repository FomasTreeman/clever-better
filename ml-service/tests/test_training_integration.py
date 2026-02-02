"""Integration tests for training pipeline."""
from __future__ import annotations

import pytest
import numpy as np
from datetime import datetime, timedelta


class TestTrainingPipeline:
    """Test end-to-end training pipeline."""
    
    @pytest.mark.asyncio
    async def test_data_loading(self):
        """Test loading data from database."""
        from app.training import TrainingPipeline, TrainingConfig
        
        config = TrainingConfig()
        pipeline = TrainingPipeline(config)
        
        # This would require database setup - mock or skip in real tests
        # data = await pipeline.load_data_from_database()
        # assert len(data) > 0
        pass
    
    def test_data_preprocessing(self):
        """Test data preprocessing and splitting."""
        from app.training import TrainingPipeline, TrainingConfig
        import pandas as pd
        
        config = TrainingConfig()
        pipeline = TrainingPipeline(config)
        
        # Create dummy data
        dates = [datetime.now() + timedelta(days=i) for i in range(100)]
        data = pd.DataFrame({
            'run_date': dates,
            'sharpe_ratio': np.random.randn(100),
            'roi': np.random.randn(100),
            'max_drawdown': np.random.rand(100),
            'win_rate': np.random.rand(100),
            'composite_score': np.random.rand(100),
            'recommendation': np.random.choice(['bet', 'no_bet'], 100)
        })
        
        X_train, X_val, X_test, y_train, y_val, y_test = pipeline.preprocess_data(data)
        
        assert len(X_train) > 0
        assert len(X_val) > 0
        assert len(X_test) > 0
        assert len(y_train) == len(X_train)
    
    @pytest.mark.asyncio
    async def test_model_training(self):
        """Test model training."""
        from app.training import TrainingPipeline, TrainingConfig
        
        config = TrainingConfig()
        pipeline = TrainingPipeline(config)
        
        # Create dummy data
        X_train = np.random.randn(100, 20)
        y_train = np.random.randint(0, 2, 100)
        X_val = np.random.randn(20, 20)
        y_val = np.random.randint(0, 2, 20)
        
        model = await pipeline.train_model(
            X_train, y_train, X_val, y_val,
            model_type='ensemble'
        )
        
        assert model is not None
    
    @pytest.mark.asyncio
    async def test_hyperparameter_search(self):
        """Test hyperparameter optimization."""
        from app.training import TrainingPipeline, TrainingConfig
        
        config = TrainingConfig()
        pipeline = TrainingPipeline(config)
        
        # Create dummy data
        X_train = np.random.randn(100, 20)
        y_train = np.random.randint(0, 2, 100)
        X_val = np.random.randn(20, 20)
        y_val = np.random.randint(0, 2, 20)
        
        best_params = await pipeline.hyperparameter_search(
            X_train, y_train, X_val, y_val,
            model_type='ensemble',
            n_trials=5  # Small number for testing
        )
        
        assert best_params is not None
        assert isinstance(best_params, dict)


class TestModelEvaluation:
    """Test model evaluation framework."""
    
    def test_classification_metrics(self):
        """Test classification metric calculation."""
        from app.evaluation import ModelEvaluator
        
        y_true = np.array([0, 1, 1, 0, 1])
        y_pred = np.array([0, 1, 0, 0, 1])
        y_pred_proba = np.array([0.1, 0.9, 0.4, 0.2, 0.8])
        
        evaluator = ModelEvaluator()
        metrics = evaluator.calculate_classification_metrics(y_true, y_pred, y_pred_proba)
        
        assert 'accuracy' in metrics
        assert 'precision' in metrics
        assert 'roc_auc' in metrics
    
    def test_betting_metrics(self):
        """Test betting metric calculation."""
        from app.evaluation import ModelEvaluator
        
        stakes = np.array([100, 100, 100, 100])
        returns = np.array([110, 90, 120, 85])
        
        evaluator = ModelEvaluator()
        metrics = evaluator.calculate_betting_metrics(stakes, returns)
        
        assert 'roi' in metrics
        assert 'sharpe_ratio' in metrics
        assert 'max_drawdown' in metrics
    
    def test_model_comparison(self):
        """Test model comparison and ranking."""
        from app.evaluation import ModelEvaluator
        
        models_metrics = {
            'model1': {
                'sharpe_ratio': 1.5,
                'roi': 0.15,
                'profit_factor': 1.8,
                'brier_score': 0.2,
                'win_rate': 0.55
            },
            'model2': {
                'sharpe_ratio': 1.2,
                'roi': 0.12,
                'profit_factor': 1.6,
                'brier_score': 0.25,
                'win_rate': 0.52
            }
        }
        
        evaluator = ModelEvaluator()
        rankings = evaluator.compare_models(models_metrics)
        
        assert 'rankings' in rankings
        assert 'best_model' in rankings
        assert rankings['best_model'] in ['model1', 'model2']
