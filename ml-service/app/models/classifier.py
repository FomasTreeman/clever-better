"""TensorFlow-based classifier for race outcome prediction."""
from __future__ import annotations

from typing import Any, Dict, List, Tuple

import numpy as np
import tensorflow as tf
from tensorflow import keras
from tensorflow.keras import layers, callbacks
from sklearn.isotonic import IsotonicRegression
from sklearn.model_selection import train_test_split


class RaceOutcomeClassifier:
    """Neural network classifier for predicting race outcomes."""
    
    def __init__(
        self,
        input_dim: int,
        learning_rate: float = 0.001,
        dropout_rate_1: float = 0.3,
        dropout_rate_2: float = 0.2,
    ):
        self.input_dim = input_dim
        self.learning_rate = learning_rate
        self.dropout_rate_1 = dropout_rate_1
        self.dropout_rate_2 = dropout_rate_2
        self.model = self._build_model()
        self.calibrator = ProbabilityCalibrator()
        self.history = None
    
    def _build_model(self) -> keras.Model:
        """Build neural network architecture."""
        model = keras.Sequential([
            layers.Input(shape=(self.input_dim,)),
            layers.Dense(128, activation='relu', kernel_regularizer=keras.regularizers.l2(0.01)),
            layers.BatchNormalization(),
            layers.Dropout(self.dropout_rate_1),
            layers.Dense(64, activation='relu', kernel_regularizer=keras.regularizers.l2(0.01)),
            layers.BatchNormalization(),
            layers.Dropout(self.dropout_rate_2),
            layers.Dense(32, activation='relu'),
            layers.Dense(1, activation='sigmoid'),
        ])
        
        # Custom loss combining binary cross-entropy with calibration penalty
        def calibrated_loss(y_true, y_pred):
            bce = keras.losses.binary_crossentropy(y_true, y_pred)
            # Calibration penalty: encourage predictions to match empirical frequencies
            calibration_penalty = tf.reduce_mean(tf.square(y_pred - y_true), axis=0)
            return bce + 0.1 * calibration_penalty
        
        model.compile(
            optimizer=keras.optimizers.Adam(learning_rate=self.learning_rate),
            loss=calibrated_loss,
            metrics=[
                'accuracy',
                keras.metrics.AUC(name='auc'),
                keras.metrics.Precision(name='precision'),
                keras.metrics.Recall(name='recall'),
            ]
        )
        
        return model
    
    def train(
        self,
        X: np.ndarray,
        y: np.ndarray,
        validation_split: float = 0.2,
        epochs: int = 100,
        batch_size: int = 32,
        early_stopping_patience: int = 10,
        verbose: int = 1,
    ) -> Dict[str, List[float]]:
        """Train the classifier with early stopping."""
        # Create validation split maintaining temporal order
        split_idx = int(len(X) * (1 - validation_split))
        x_train, x_val = X[:split_idx], X[split_idx:]
        y_train, y_val = y[:split_idx], y[split_idx:]
        
        # Define callbacks
        early_stop = callbacks.EarlyStopping(
            monitor='val_loss',
            patience=early_stopping_patience,
            restore_best_weights=True,
            verbose=verbose
        )
        
        reduce_lr = callbacks.ReduceLROnPlateau(
            monitor='val_loss',
            factor=0.5,
            patience=5,
            min_lr=1e-6,
            verbose=verbose
        )
        
        # Train model
        self.history = self.model.fit(
            x_train, y_train,
            validation_data=(x_val, y_val),
            epochs=epochs,
            batch_size=batch_size,
            callbacks=[early_stop, reduce_lr],
            verbose=verbose
        )
        
        # Train calibrator on validation set
        val_predictions = self.model.predict(x_val, verbose=0).flatten()
        self.calibrator.fit(val_predictions, y_val)
        
        return self.history.history
    
    def predict_probabilities(
        self,
        X: np.ndarray,
        calibrated: bool = True
    ) -> np.ndarray:
        """Generate predictions with optional calibration."""
        raw_predictions = self.model.predict(X, verbose=0).flatten()
        
        if calibrated and self.calibrator.is_fitted:
            return self.calibrator.transform(raw_predictions)
        
        return raw_predictions
    
    def evaluate(self, X: np.ndarray, y: np.ndarray) -> Dict[str, float]:
        """Evaluate model performance."""
        predictions = self.predict_probabilities(X, calibrated=True)
        
        # Calculate Brier score
        brier_score = np.mean((predictions - y) ** 2)
        
        # Calculate log loss
        epsilon = 1e-15
        predictions_clipped = np.clip(predictions, epsilon, 1 - epsilon)
        log_loss = -np.mean(y * np.log(predictions_clipped) + (1 - y) * np.log(1 - predictions_clipped))
        
        # Get model metrics
        metrics = self.model.evaluate(X, y, verbose=0, return_dict=True)
        
        metrics['brier_score'] = float(brier_score)
        metrics['log_loss'] = float(log_loss)
        
        return metrics
    
    def save(self, model_path: str, calibrator_path: str):
        """Save model and calibrator."""
        self.model.save(model_path)
        if self.calibrator.is_fitted:
            import joblib
            joblib.dump(self.calibrator, calibrator_path)
    
    def load(self, model_path: str, calibrator_path: str):
        """Load model and calibrator."""
        self.model = keras.models.load_model(model_path)
        try:
            import joblib
            self.calibrator = joblib.load(calibrator_path)
        except FileNotFoundError:
            self.calibrator = ProbabilityCalibrator()


class ProbabilityCalibrator:
    """Isotonic regression for probability calibration."""
    
    def __init__(self):
        self.calibrator = IsotonicRegression(out_of_bounds='clip')
        self.is_fitted = False
    
    def fit(self, predictions: np.ndarray, targets: np.ndarray):
        """Fit calibrator on predictions and targets."""
        self.calibrator.fit(predictions, targets)
        self.is_fitted = True
    
    def transform(self, predictions: np.ndarray) -> np.ndarray:
        """Transform predictions using fitted calibrator."""
        if not self.is_fitted:
            return predictions
        return self.calibrator.transform(predictions)


def train_classifier(
    X: np.ndarray,
    y: np.ndarray,
    input_dim: int,
    **kwargs
) -> RaceOutcomeClassifier:
    """Train classifier with specified configuration."""
    classifier = RaceOutcomeClassifier(input_dim=input_dim, **kwargs)
    classifier.train(X, y)
    return classifier


def calculate_calibration_metrics(
    predictions: np.ndarray,
    targets: np.ndarray,
    n_bins: int = 10
) -> Dict[str, Any]:
    """Calculate calibration metrics and bin statistics."""
    bin_edges = np.linspace(0, 1, n_bins + 1)
    bin_indices = np.digitize(predictions, bin_edges[1:-1])
    
    bin_stats = []
    for i in range(n_bins):
        mask = bin_indices == i
        if mask.sum() > 0:
            bin_predictions = predictions[mask]
            bin_targets = targets[mask]
            
            bin_stats.append({
                'bin_index': i,
                'count': int(mask.sum()),
                'mean_prediction': float(bin_predictions.mean()),
                'mean_target': float(bin_targets.mean()),
                'calibration_error': float(abs(bin_predictions.mean() - bin_targets.mean())),
            })
    
    # Expected Calibration Error (ECE)
    ece = sum(
        stat['count'] / len(predictions) * stat['calibration_error']
        for stat in bin_stats
    )
    
    # Maximum Calibration Error (MCE)
    mce = max((stat['calibration_error'] for stat in bin_stats), default=0)
    
    return {
        'expected_calibration_error': ece,
        'maximum_calibration_error': mce,
        'bin_statistics': bin_stats,
    }
