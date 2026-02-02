"""Unit tests for ML models."""
from __future__ import annotations

import pytest
import numpy as np
import torch


class TestRLAgent:
    """Test RL agent implementation."""
    
    def test_betting_environment_initialization(self):
        """Test environment initializes correctly."""
        from app.models.rl_agent import BettingEnvironment
        
        env = BettingEnvironment(initial_capital=1000.0)
        state = env.reset()
        
        assert env.capital == 1000.0
        assert len(state) == env.state_size
        assert env.episode_length == 100
    
    def test_betting_environment_step(self):
        """Test environment step function."""
        from app.models.rl_agent import BettingEnvironment
        
        env = BettingEnvironment(initial_capital=1000.0)
        env.reset()
        
        # Take action (bet 10% of Kelly)
        action = 1
        next_state, reward, done = env.step(action)
        
        assert len(next_state) == env.state_size
        assert isinstance(reward, float)
        assert isinstance(done, bool)
    
    def test_policy_network_forward(self):
        """Test policy network forward pass."""
        from app.models.rl_agent import PolicyNetwork
        
        state_size = 10
        action_size = 11
        network = PolicyNetwork(state_size, action_size)
        
        state = torch.randn(1, state_size)
        output = network(state)
        
        assert output.shape == (1, action_size)
    
    def test_dqn_agent_select_action(self):
        """Test DQN agent action selection."""
        from app.models.rl_agent import DQNAgent
        
        state_size = 10
        action_size = 11
        agent = DQNAgent(state_size, action_size)
        
        state = np.random.randn(state_size)
        action = agent.select_action(state)
        
        assert 0 <= action < action_size


class TestClassifier:
    """Test TensorFlow classifier."""
    
    def test_classifier_build(self):
        """Test classifier model builds correctly."""
        from app.models.classifier import RaceOutcomeClassifier
        
        input_dim = 20
        classifier = RaceOutcomeClassifier(input_dim=input_dim)
        model = classifier.build_model()
        
        assert model is not None
        assert len(model.layers) > 0
    
    def test_classifier_train(self):
        """Test classifier training."""
        from app.models.classifier import RaceOutcomeClassifier
        
        # Create dummy data
        X_train = np.random.randn(100, 20)
        y_train = np.random.randint(0, 2, 100)
        X_val = np.random.randn(20, 20)
        y_val = np.random.randint(0, 2, 20)
        
        classifier = RaceOutcomeClassifier(input_dim=20)
        model, history = classifier.train(X_train, y_train, X_val, y_val, epochs=2)
        
        assert model is not None
        assert 'loss' in history.history
    
    def test_probability_calibrator(self):
        """Test probability calibration."""
        from app.models.classifier import ProbabilityCalibrator
        
        # Create dummy probabilities and labels
        y_true = np.random.randint(0, 2, 100)
        y_pred_proba = np.random.rand(100)
        
        calibrator = ProbabilityCalibrator()
        calibrator.fit(y_true, y_pred_proba)
        
        calibrated = calibrator.calibrate(y_pred_proba)
        
        assert len(calibrated) == len(y_pred_proba)
        assert np.all((calibrated >= 0) & (calibrated <= 1))


class TestEnsemble:
    """Test ensemble models."""
    
    def test_ensemble_initialization(self):
        """Test ensemble initializes correctly."""
        from app.models.ensemble import StrategyEnsemble
        
        ensemble = StrategyEnsemble(ensemble_type='voting')
        
        assert ensemble.ensemble_type == 'voting'
        assert len(ensemble.base_models) == 4
    
    def test_ensemble_fit(self):
        """Test ensemble training."""
        from app.models.ensemble import StrategyEnsemble
        
        # Create dummy data
        X = np.random.randn(100, 20)
        y = np.random.randint(0, 2, 100)
        
        ensemble = StrategyEnsemble(ensemble_type='voting')
        ensemble.fit(X, y)
        
        assert ensemble.model is not None
    
    def test_ensemble_predict(self):
        """Test ensemble prediction."""
        from app.models.ensemble import StrategyEnsemble
        
        # Create and train ensemble
        X = np.random.randn(100, 20)
        y = np.random.randint(0, 2, 100)
        
        ensemble = StrategyEnsemble(ensemble_type='voting')
        ensemble.fit(X, y)
        
        # Predict
        X_test = np.random.randn(10, 20)
        predictions = ensemble.predict(X_test)
        
        assert len(predictions) == 10
    
    def test_feature_importance(self):
        """Test feature importance extraction."""
        from app.models.ensemble import StrategyEnsemble
        
        X = np.random.randn(100, 20)
        y = np.random.randint(0, 2, 100)
        
        ensemble = StrategyEnsemble(ensemble_type='voting')
        ensemble.fit(X, y)
        
        importance = ensemble.get_feature_importance()
        
        assert len(importance) == 4  # One per base model
