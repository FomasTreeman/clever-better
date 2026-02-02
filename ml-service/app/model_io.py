"""Model persistence utilities."""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Dict
import joblib
import torch


@dataclass
class ModelArtifact:
    """Container for model with metadata."""
    model: Any
    metadata: Dict[str, Any]
    version: str


def save_pytorch_model(model: Any, path: str, optimizer: Any = None, config: Dict = None):
    """Save PyTorch model."""
    checkpoint = {
        'model_state_dict': model.state_dict(),
        'config': config or {},
    }
    if optimizer:
        checkpoint['optimizer_state_dict'] = optimizer.state_dict()
    
    torch.save(checkpoint, path)


def load_pytorch_model(model_class: Any, path: str) -> Any:
    """Load PyTorch model."""
    checkpoint = torch.load(path)
    model = model_class(**checkpoint.get('config', {}))
    model.load_state_dict(checkpoint['model_state_dict'])
    return model


def save_tensorflow_model(model: Any, path: str):
    """Save TensorFlow model."""
    model.save(path)


def save_sklearn_model(model: Any, path: str):
    """Save sklearn model."""
    joblib.dump(model, path)


def load_sklearn_model(path: str) -> Any:
    """Load sklearn model."""
    return joblib.load(path)


def save_dqn_agent(agent: Any, path: str):
    """Save DQN agent checkpoint."""
    agent.save(path)


def load_dqn_agent(agent_class: Any, path: str, **kwargs) -> Any:
    """Load DQN agent checkpoint."""
    agent = agent_class(**kwargs)
    agent.load(path)
    return agent
