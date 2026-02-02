"""Ensemble methods using scikit-learn, XGBoost, and LightGBM."""
from __future__ import annotations

from typing import Any, Dict, List, Tuple

import numpy as np
import shap
from sklearn.ensemble import (
    RandomForestClassifier,
    GradientBoostingClassifier,
    StackingClassifier,
    VotingClassifier,
)
from sklearn.linear_model import LogisticRegression
from xgboost import XGBClassifier
from lightgbm import LGBMClassifier
import joblib


class StrategyEnsemble:
    """Ensemble combining multiple base models for strategy prediction."""
    
    def __init__(self, use_stacking: bool = True):
        self.use_stacking = use_stacking
        self.base_models = self._create_base_models()
        self.ensemble = None
        self.feature_names = None
        self.is_fitted = False
    
    def _create_base_models(self) -> Dict[str, Any]:
        """Create base classifier models."""
        return {
            'random_forest': RandomForestClassifier(
                n_estimators=500,
                max_depth=10,
                min_samples_split=20,
                random_state=42,
                n_jobs=-1,
                class_weight='balanced'
            ),
            'gradient_boosting': GradientBoostingClassifier(
                n_estimators=300,
                learning_rate=0.05,
                max_depth=6,
                random_state=42,
                subsample=0.8
            ),
            'xgboost': XGBClassifier(
                n_estimators=500,
                learning_rate=0.05,
                max_depth=6,
                subsample=0.8,
                random_state=42,
                eval_metric='logloss',
                use_label_encoder=False
            ),
            'lightgbm': LGBMClassifier(
                n_estimators=500,
                learning_rate=0.05,
                num_leaves=31,
                random_state=42,
                verbose=-1
            ),
        }
    
    def train(self, X: np.ndarray, y: np.ndarray, feature_names: List[str] = None):
        """Train ensemble models."""
        self.feature_names = feature_names
        
        if self.use_stacking:
            # Create stacking ensemble with logistic regression meta-learner
            estimators = [(name, model) for name, model in self.base_models.items()]
            self.ensemble = StackingClassifier(
                estimators=estimators,
                final_estimator=LogisticRegression(max_iter=1000, random_state=42),
                cv=5,
                n_jobs=-1
            )
        else:
            # Create voting ensemble with soft voting
            estimators = [(name, model) for name, model in self.base_models.items()]
            self.ensemble = VotingClassifier(
                estimators=estimators,
                voting='soft',
                n_jobs=-1
            )
        
        self.ensemble.fit(X, y)
        self.is_fitted = True
    
    def predict(self, X: np.ndarray) -> np.ndarray:
        """Generate predictions."""
        if not self.is_fitted:
            raise ValueError("Ensemble must be fitted before making predictions")
        return self.ensemble.predict(X)
    
    def predict_proba(self, X: np.ndarray) -> np.ndarray:
        """Generate probability predictions."""
        if not self.is_fitted:
            raise ValueError("Ensemble must be fitted before making predictions")
        return self.ensemble.predict_proba(X)[:, 1]
    
    def predict_with_confidence(
        self,
        X: np.ndarray,
        confidence_method: str = 'variance'
    ) -> Tuple[np.ndarray, np.ndarray]:
        """Generate predictions with confidence intervals."""
        if not self.is_fitted:
            raise ValueError("Ensemble must be fitted before making predictions")
        
        # Get predictions from all base models
        base_predictions = []
        for name, model in self.base_models.items():
            if hasattr(self.ensemble, 'named_estimators_'):
                fitted_model = self.ensemble.named_estimators_[name]
            else:
                fitted_model = model
            
            if hasattr(fitted_model, 'predict_proba'):
                pred_proba = fitted_model.predict_proba(X)[:, 1]
            else:
                pred_proba = fitted_model.predict(X)
            
            base_predictions.append(pred_proba)
        
        base_predictions = np.array(base_predictions)
        
        # Calculate ensemble prediction
        predictions = self.ensemble.predict_proba(X)[:, 1]
        
        # Calculate confidence based on variance or std
        if confidence_method == 'variance':
            confidence = 1 - np.var(base_predictions, axis=0)
        else:  # std
            confidence = 1 - np.std(base_predictions, axis=0)
        
        # Normalize confidence to [0, 1]
        confidence = np.clip(confidence, 0, 1)
        
        return predictions, confidence
    
    def get_feature_importance(
        self,
        method: str = 'aggregate'
    ) -> Dict[str, float]:
        """Extract feature importance from tree-based models."""
        if not self.is_fitted:
            raise ValueError("Ensemble must be fitted before extracting feature importance")
        
        importance_dict = {}
        
        for name, model in self.base_models.items():
            if hasattr(self.ensemble, 'named_estimators_'):
                fitted_model = self.ensemble.named_estimators_[name]
            else:
                fitted_model = model
            
            if hasattr(fitted_model, 'feature_importances_'):
                importances = fitted_model.feature_importances_
                
                if self.feature_names:
                    for feat_name, importance in zip(self.feature_names, importances):
                        if feat_name not in importance_dict:
                            importance_dict[feat_name] = []
                        importance_dict[feat_name].append(importance)
        
        # Aggregate importances
        if method == 'aggregate':
            aggregated = {
                feat: np.mean(imps) for feat, imps in importance_dict.items()
            }
        elif method == 'max':
            aggregated = {
                feat: np.max(imps) for feat, imps in importance_dict.items()
            }
        else:  # sum
            aggregated = {
                feat: np.sum(imps) for feat, imps in importance_dict.items()
            }
        
        # Sort by importance
        sorted_importance = dict(
            sorted(aggregated.items(), key=lambda x: x[1], reverse=True)
        )
        
        return sorted_importance
    
    def calculate_shap_values(
        self,
        X: np.ndarray,
        sample_size: int = 100
    ) -> Dict[str, Any]:
        """Calculate SHAP values for model interpretability."""
        if not self.is_fitted:
            raise ValueError("Ensemble must be fitted before calculating SHAP values")
        
        # Sample data for SHAP calculation (expensive operation)
        if len(X) > sample_size:
            indices = np.random.choice(len(X), sample_size, replace=False)
            X_sample = X[indices]
        else:
            X_sample = X
        
        shap_values_dict = {}
        
        # Calculate SHAP values for tree-based models
        for name in ['random_forest', 'xgboost', 'lightgbm']:
            if name in self.base_models:
                if hasattr(self.ensemble, 'named_estimators_'):
                    model = self.ensemble.named_estimators_[name]
                else:
                    model = self.base_models[name]
                
                try:
                    explainer = shap.TreeExplainer(model)
                    shap_values = explainer.shap_values(X_sample)
                    
                    # Handle binary classification output
                    if isinstance(shap_values, list):
                        shap_values = shap_values[1]  # Get positive class
                    
                    shap_values_dict[name] = shap_values
                except Exception as e:
                    print(f"Error calculating SHAP for {name}: {e}")
        
        # Aggregate SHAP values across models
        if shap_values_dict:
            aggregated_shap = np.mean(list(shap_values_dict.values()), axis=0)
            
            # Calculate mean absolute SHAP values for feature importance
            mean_abs_shap = np.abs(aggregated_shap).mean(axis=0)
            
            if self.feature_names:
                feature_importance = dict(zip(self.feature_names, mean_abs_shap))
                feature_importance = dict(
                    sorted(feature_importance.items(), key=lambda x: x[1], reverse=True)
                )
            else:
                feature_importance = {f'feature_{i}': val for i, val in enumerate(mean_abs_shap)}
            
            return {
                'shap_values': aggregated_shap,
                'feature_importance': feature_importance,
                'base_values': explainer.expected_value if 'explainer' in locals() else 0,
            }
        
        return {'shap_values': None, 'feature_importance': {}, 'base_values': 0}
    
    def save(self, path: str):
        """Save ensemble model."""
        model_data = {
            'ensemble': self.ensemble,
            'base_models': self.base_models,
            'feature_names': self.feature_names,
            'use_stacking': self.use_stacking,
            'is_fitted': self.is_fitted,
        }
        joblib.dump(model_data, path)
    
    def load(self, path: str):
        """Load ensemble model."""
        model_data = joblib.load(path)
        self.ensemble = model_data['ensemble']
        self.base_models = model_data['base_models']
        self.feature_names = model_data['feature_names']
        self.use_stacking = model_data['use_stacking']
        self.is_fitted = model_data['is_fitted']


def train_ensemble(
    X: np.ndarray,
    y: np.ndarray,
    feature_names: List[str] = None,
    use_stacking: bool = True
) -> StrategyEnsemble:
    """Train ensemble model with specified configuration."""
    ensemble = StrategyEnsemble(use_stacking=use_stacking)
    ensemble.train(X, y, feature_names=feature_names)
    return ensemble
