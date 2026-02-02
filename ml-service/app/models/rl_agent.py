"""Reinforcement Learning agent for betting strategy optimization using PyTorch."""
from __future__ import annotations

import random
from collections import deque
from dataclasses import dataclass
from typing import Any, Dict, List, Tuple

import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim


@dataclass
class BettingState:
    """State representation for betting environment."""
    current_capital: float
    recent_returns: List[float]
    recent_sharpe: float
    recent_drawdown: float
    market_avg_odds: float
    market_volume: float
    strategy_win_rate: float
    strategy_profit_factor: float
    risk_var_95: float


class BettingEnvironment:
    """Simulates betting decisions using historical backtest data."""
    
    def __init__(self, backtest_results: List[Dict[str, Any]]):
        self.backtest_results = backtest_results
        self.current_index = 0
        self.current_capital = 10000.0
        self.initial_capital = 10000.0
        self.equity_history = [self.current_capital]
        self.action_history = []
        
    def reset(self) -> np.ndarray:
        """Reset environment to initial state."""
        self.current_index = 0
        self.current_capital = self.initial_capital
        self.equity_history = [self.current_capital]
        self.action_history = []
        return self._get_state()
    
    def _get_state(self) -> np.ndarray:
        """Extract state representation from current position."""
        if self.current_index >= len(self.backtest_results):
            return np.zeros(9)
        
        result = self.backtest_results[self.current_index]
        recent_equity = self.equity_history[-10:] if len(self.equity_history) >= 10 else self.equity_history
        recent_returns = np.diff(recent_equity) / np.array(recent_equity[:-1]) if len(recent_equity) > 1 else [0]
        
        # Calculate recent Sharpe ratio
        if len(recent_returns) > 1:
            recent_sharpe = np.mean(recent_returns) / (np.std(recent_returns) + 1e-6)
        else:
            recent_sharpe = 0.0
        
        # Calculate recent drawdown
        peak = max(recent_equity)
        recent_drawdown = (peak - self.current_capital) / peak if peak > 0 else 0
        
        # Extract market features
        full_results = result.get('full_results', {})
        market_features = full_results.get('market', {})
        
        state = np.array([
            self.current_capital / self.initial_capital,  # Normalized capital
            np.mean(recent_returns) if len(recent_returns) > 0 else 0,
            recent_sharpe,
            recent_drawdown,
            market_features.get('avg_odds', 0),
            market_features.get('avg_volume', 0),
            result.get('win_rate', 0),
            result.get('profit_factor', 0),
            full_results.get('risk_profile', {}).get('var_95', 0),
        ], dtype=np.float32)
        
        return state
    
    def step(self, action: int) -> Tuple[np.ndarray, float, bool, Dict]:
        """Execute action and return new state, reward, done, info."""
        if self.current_index >= len(self.backtest_results):
            return self._get_state(), 0.0, True, {}
        
        result = self.backtest_results[self.current_index]
        
        # Decode action: [0-10] = bet size as % of Kelly (0%, 10%, ..., 100%)
        bet_fraction = action / 10.0
        
        # Calculate position size using Kelly fraction
        kelly_fraction = result.get('profit_factor', 1.0) - 1.0
        kelly_fraction = max(0, min(kelly_fraction, 0.25))  # Cap at 25%
        position_size = self.current_capital * kelly_fraction * bet_fraction
        
        # Simulate bet outcome based on backtest return
        bet_return = result.get('total_return', 0.0)
        pnl = position_size * bet_return
        self.current_capital += pnl
        
        self.equity_history.append(self.current_capital)
        self.action_history.append(action)
        
        # Calculate reward: risk-adjusted return (Sharpe weighted with profit factor)
        sharpe = result.get('sharpe_ratio', 0.0)
        profit_factor = result.get('profit_factor', 1.0)
        reward = sharpe * np.log(profit_factor + 1e-6)
        
        # Penalty for excessive drawdown
        if self.current_capital < self.initial_capital * 0.7:
            reward -= 10.0
        
        # Bonus for capital growth
        if self.current_capital > max(self.equity_history[:-1]):
            reward += 1.0
        
        self.current_index += 1
        done = self.current_index >= len(self.backtest_results)
        
        info = {
            'capital': self.current_capital,
            'position_size': position_size,
            'pnl': pnl,
            'sharpe': sharpe,
        }
        
        return self._get_state(), reward, done, info


class PolicyNetwork(nn.Module):
    """Neural network for Q-value estimation."""
    
    def __init__(self, state_dim: int, action_dim: int):
        super().__init__()
        self.network = nn.Sequential(
            nn.Linear(state_dim, 256),
            nn.ReLU(),
            nn.Linear(256, 128),
            nn.ReLU(),
            nn.Linear(128, 64),
            nn.ReLU(),
            nn.Linear(64, action_dim)
        )
    
    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.network(x)


class DQNAgent:
    """Deep Q-Network agent with experience replay."""
    
    def __init__(
        self,
        state_dim: int,
        action_dim: int,
        learning_rate: float = 0.001,
        gamma: float = 0.99,
        epsilon_start: float = 1.0,
        epsilon_end: float = 0.01,
        epsilon_decay: float = 0.995,
        buffer_size: int = 10000,
        batch_size: int = 64,
        target_update_freq: int = 100,
    ):
        self.state_dim = state_dim
        self.action_dim = action_dim
        self.gamma = gamma
        self.epsilon = epsilon_start
        self.epsilon_end = epsilon_end
        self.epsilon_decay = epsilon_decay
        self.batch_size = batch_size
        self.target_update_freq = target_update_freq
        self.steps = 0
        
        # Networks
        self.policy_net = PolicyNetwork(state_dim, action_dim)
        self.target_net = PolicyNetwork(state_dim, action_dim)
        self.target_net.load_state_dict(self.policy_net.state_dict())
        self.target_net.eval()
        
        self.optimizer = optim.Adam(self.policy_net.parameters(), lr=learning_rate)
        self.loss_fn = nn.SmoothL1Loss()
        
        # Experience replay buffer
        self.replay_buffer = deque(maxlen=buffer_size)
    
    def select_action(self, state: np.ndarray, training: bool = True) -> int:
        """Select action using epsilon-greedy policy."""
        if training and random.random() < self.epsilon:
            return random.randint(0, self.action_dim - 1)
        
        with torch.no_grad():
            state_tensor = torch.FloatTensor(state).unsqueeze(0)
            q_values = self.policy_net(state_tensor)
            return q_values.argmax().item()
    
    def store_transition(
        self,
        state: np.ndarray,
        action: int,
        reward: float,
        next_state: np.ndarray,
        done: bool
    ):
        """Store transition in replay buffer."""
        self.replay_buffer.append((state, action, reward, next_state, done))
    
    def train_step(self) -> float:
        """Perform one training step."""
        if len(self.replay_buffer) < self.batch_size:
            return 0.0
        
        # Sample batch from replay buffer
        batch = random.sample(self.replay_buffer, self.batch_size)
        states, actions, rewards, next_states, dones = zip(*batch)
        
        states = torch.FloatTensor(np.array(states))
        actions = torch.LongTensor(actions)
        rewards = torch.FloatTensor(rewards)
        next_states = torch.FloatTensor(np.array(next_states))
        dones = torch.FloatTensor(dones)
        
        # Compute Q-values
        current_q_values = self.policy_net(states).gather(1, actions.unsqueeze(1))
        
        # Compute target Q-values
        with torch.no_grad():
            next_q_values = self.target_net(next_states).max(1)[0]
            target_q_values = rewards + (1 - dones) * self.gamma * next_q_values
        
        # Compute loss and update
        loss = self.loss_fn(current_q_values.squeeze(), target_q_values)
        
        self.optimizer.zero_grad()
        loss.backward()
        torch.nn.utils.clip_grad_norm_(self.policy_net.parameters(), 1.0)
        self.optimizer.step()
        
        # Update target network
        self.steps += 1
        if self.steps % self.target_update_freq == 0:
            self.target_net.load_state_dict(self.policy_net.state_dict())
        
        # Decay epsilon
        self.epsilon = max(self.epsilon_end, self.epsilon * self.epsilon_decay)
        
        return loss.item()
    
    def save(self, path: str):
        """Save agent state."""
        torch.save({
            'policy_net': self.policy_net.state_dict(),
            'target_net': self.target_net.state_dict(),
            'optimizer': self.optimizer.state_dict(),
            'epsilon': self.epsilon,
            'steps': self.steps,
            'replay_buffer': list(self.replay_buffer),
        }, path)
    
    def load(self, path: str):
        """Load agent state."""
        checkpoint = torch.load(path)
        self.policy_net.load_state_dict(checkpoint['policy_net'])
        self.target_net.load_state_dict(checkpoint['target_net'])
        self.optimizer.load_state_dict(checkpoint['optimizer'])
        self.epsilon = checkpoint['epsilon']
        self.steps = checkpoint['steps']
        self.replay_buffer = deque(checkpoint['replay_buffer'], maxlen=self.replay_buffer.maxlen)


def train_rl_agent(
    backtest_results: List[Dict[str, Any]],
    num_episodes: int = 1000,
    max_steps: int = 500,
) -> DQNAgent:
    """Train RL agent on backtest results."""
    env = BettingEnvironment(backtest_results)
    agent = DQNAgent(state_dim=9, action_dim=11)
    
    episode_rewards = []
    episode_losses = []
    
    for episode in range(num_episodes):
        state = env.reset()
        episode_reward = 0
        episode_loss = 0
        steps = 0
        
        for step in range(max_steps):
            action = agent.select_action(state, training=True)
            next_state, reward, done, info = env.step(action)
            
            agent.store_transition(state, action, reward, next_state, done)
            loss = agent.train_step()
            
            episode_reward += reward
            episode_loss += loss
            steps += 1
            state = next_state
            
            if done:
                break
        
        episode_rewards.append(episode_reward)
        episode_losses.append(episode_loss / steps if steps > 0 else 0)
        
        if (episode + 1) % 100 == 0:
            avg_reward = np.mean(episode_rewards[-100:])
            avg_loss = np.mean(episode_losses[-100:])
            print(f"Episode {episode + 1}/{num_episodes}, Avg Reward: {avg_reward:.2f}, Avg Loss: {avg_loss:.4f}, Epsilon: {agent.epsilon:.3f}")
    
    return agent
