package backtest

import (
	"fmt"
	"time"

	"github.com/yourusername/clever-better/internal/config"
)

// BacktestConfig extends core config with backtest-specific settings
type BacktestConfig struct {
	StartDate            time.Time
	EndDate              time.Time
	InitialBankroll      float64
	CommissionRate       float64
	SlippageTicks        int
	MinLiquidity         float64
	OutputPath           string
	MLExportEnabled      bool
	MonteCarloIterations int
	WalkForwardWindows   int
	RiskFreeRate         float64
}

// FromConfig converts app config to backtest config
func FromConfig(cfg *config.BacktestConfig) (BacktestConfig, error) {
	if cfg == nil {
		return BacktestConfig{}, fmt.Errorf("backtest config is required")
	}
	start, err := time.Parse("2006-01-02", cfg.StartDate)
	if err != nil {
		return BacktestConfig{}, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", cfg.EndDate)
	if err != nil {
		return BacktestConfig{}, fmt.Errorf("invalid end date: %w", err)
	}

	bt := BacktestConfig{
		StartDate:            start,
		EndDate:              end,
		InitialBankroll:      cfg.InitialBankroll,
		CommissionRate:       cfg.CommissionRate,
		SlippageTicks:        cfg.SlippageTicks,
		MinLiquidity:         cfg.MinLiquidity,
		OutputPath:           cfg.OutputPath,
		MLExportEnabled:      cfg.MLExportEnabled,
		MonteCarloIterations: cfg.MonteCarloIterations,
		WalkForwardWindows:   cfg.WalkForwardWindows,
		RiskFreeRate:         cfg.RiskFreeRate,
	}

	return bt, bt.Validate()
}

// Validate validates backtest config parameters
func (b BacktestConfig) Validate() error {
	if b.StartDate.After(b.EndDate) {
		return fmt.Errorf("start date must be before end date")
	}
	if b.InitialBankroll <= 0 {
		return fmt.Errorf("initial bankroll must be positive")
	}
	if b.CommissionRate < 0 || b.CommissionRate > 0.1 {
		return fmt.Errorf("commission rate must be between 0 and 0.1")
	}
	if b.SlippageTicks < 0 {
		return fmt.Errorf("slippage ticks cannot be negative")
	}
	if b.MonteCarloIterations <= 0 {
		return fmt.Errorf("monte carlo iterations must be positive")
	}
	return nil
}
