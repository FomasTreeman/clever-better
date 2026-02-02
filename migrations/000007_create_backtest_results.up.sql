-- Create backtest results table

CREATE TABLE IF NOT EXISTS backtest_results (
    id UUID PRIMARY KEY,
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    run_date TIMESTAMPTZ NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    initial_capital DECIMAL(12, 2) NOT NULL,
    final_capital DECIMAL(12, 2) NOT NULL,
    total_return DECIMAL(8, 4) NOT NULL,
    sharpe_ratio DECIMAL(8, 4) NOT NULL,
    max_drawdown DECIMAL(8, 4) NOT NULL,
    total_bets INT NOT NULL,
    win_rate DECIMAL(8, 4) NOT NULL,
    profit_factor DECIMAL(8, 4) NOT NULL,
    method TEXT NOT NULL,
    composite_score DECIMAL(8, 4) NOT NULL,
    recommendation TEXT NOT NULL,
    ml_features JSONB,
    full_results JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backtest_results_strategy_id ON backtest_results(strategy_id, run_date DESC);
CREATE INDEX idx_backtest_results_run_date ON backtest_results(run_date DESC);
