// Package main provides the entry point for the backtesting CLI tool.
package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/backtest"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/strategy"
)

func main() {
	var (
		configPath = flag.String("config", "config/config.yaml", "Path to config file")
		strategyName = flag.String("strategy", "simple_value", "Strategy name to test")
		startDate = flag.String("start-date", "", "Override start date (YYYY-MM-DD)")
		endDate = flag.String("end-date", "", "Override end date (YYYY-MM-DD)")
		mode = flag.String("mode", "all", "Backtest mode: historical, monte-carlo, walk-forward, all")
		output = flag.String("output", "./output/backtest_results.json", "Output path for results")
		mlExport = flag.Bool("ml-export", false, "Enable ML export")
	)
	flag.Parse()

	logger := newLogger()
	ctx := context.Background()

	cfg := loadConfigWithSecrets(*configPath, logger)
	btConfig := buildBacktestConfig(cfg, *output, *mlExport, *startDate, *endDate, logger)
	strat := resolveStrategy(*strategyName)
	engine := buildEngine(ctx, cfg, btConfig, strat, logger)
	defer engine.Close(ctx)

	logger.WithFields(logrus.Fields{"mode": *mode, "strategy": strat.Name()}).Info("Starting backtest")
	runMode(ctx, engine, btConfig, strat, *mode)
}

func resolveStrategy(name string) strategy.Strategy {
	constructors := map[string]func() strategy.Strategy{
		"simple_value": strategy.NewSimpleValueStrategy,
	}
	if build, ok := constructors[name]; ok {
		return build()
	}
	return strategy.NewSimpleValueStrategy()
}

func newLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	return logger
}

func loadConfigWithSecrets(path string, logger *logrus.Logger) *config.Config {
	cfg, err := config.Load(path)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	if os.Getenv("AWS_SECRETS_ENABLED") == "true" {
		region := os.Getenv("AWS_REGION")
		secretName := os.Getenv("AWS_SECRET_NAME")
		if region == "" || secretName == "" {
			logger.Fatalf("AWS_REGION and AWS_SECRET_NAME environment variables must be set when AWS_SECRETS_ENABLED is true")
		}
		if err := config.LoadSecretsFromAWS(cfg, region, secretName); err != nil {
			logger.Fatalf("Failed to load secrets: %v", err)
		}
	}
	if err := config.Validate(cfg); err != nil {
		logger.Fatalf("Invalid configuration: %v", err)
	}
	return cfg
}

func buildBacktestConfig(cfg *config.Config, output string, mlExport bool, startOverride string, endOverride string, logger *logrus.Logger) backtest.BacktestConfig {
	btConfig, err := backtest.FromConfig(&cfg.Backtest)
	if err != nil {
		logger.Fatalf("Invalid backtest config: %v", err)
	}
	if output != "" {
		btConfig.OutputPath = output
	}
	if mlExport {
		btConfig.MLExportEnabled = true
	}
	if startOverride != "" {
		parsed, err := time.Parse("2006-01-02", startOverride)
		if err != nil {
			logger.Fatalf("Invalid start date: %v", err)
		}
		btConfig.StartDate = parsed
	}
	if endOverride != "" {
		parsed, err := time.Parse("2006-01-02", endOverride)
		if err != nil {
			logger.Fatalf("Invalid end date: %v", err)
		}
		btConfig.EndDate = parsed
	}
	return btConfig
}

func buildEngine(ctx context.Context, cfg *config.Config, btConfig backtest.BacktestConfig, strat strategy.Strategy, logger *logrus.Logger) *backtest.Engine {
	db, err := database.NewDB(ctx, &cfg.Database)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	engine, err := backtest.NewEngine(btConfig, db, strat, logger)
	if err != nil {
		logger.Fatalf("Failed to create engine: %v", err)
	}
	return engine
}

func runMode(ctx context.Context, engine *backtest.Engine, cfg backtest.BacktestConfig, strat strategy.Strategy, mode string) {
	switch mode {
	case "historical":
		runHistoricalBacktest(ctx, engine)
	case "monte-carlo":
		runMonteCarloBacktest(ctx, engine, cfg)
	case "walk-forward":
		runWalkForwardBacktest(ctx, engine, strat)
	case "all":
		runAllMethods(ctx, engine, cfg, strat)
	default:
		engineLogger(engine).Fatalf("Unsupported mode: %s", mode)
	}
}

func runHistoricalBacktest(ctx context.Context, engine *backtest.Engine) {
	state, metrics, err := engine.Run(ctx, engineConfigStart(engine), engineConfigEnd(engine))
	if err != nil {
		engineLogger(engine).Fatalf("Historical backtest failed: %v", err)
	}
	aggregated := backtest.AggregateResults(metrics, backtest.MonteCarloResult{}, backtest.WalkForwardResult{}, backtest.AggregationWeights{})
	report := backtest.GenerateConsoleReport(aggregated)
	engineLogger(engine).Info(report)
	_ = state
}

func runMonteCarloBacktest(ctx context.Context, engine *backtest.Engine, cfg backtest.BacktestConfig) {
	state, _, err := engine.Run(ctx, engineConfigStart(engine), engineConfigEnd(engine))
	if err != nil {
		engineLogger(engine).Fatalf("Historical run for Monte Carlo failed: %v", err)
	}
	probabilities := map[string]float64{}
	for _, bet := range state.Bets {
		probabilities[bet.ID.String()] = 0.5
	}
	result, err := backtest.RunMonteCarlo(ctx, state.Bets, probabilities, backtest.MonteCarloConfig{
		Iterations:      cfg.MonteCarloIterations,
		CommissionRate:  cfg.CommissionRate,
		InitialBankroll: cfg.InitialBankroll,
	})
	if err != nil {
		engineLogger(engine).Fatalf("Monte Carlo failed: %v", err)
	}
	engineLogger(engine).WithField("mean_return", result.MeanReturn).Info("Monte Carlo completed")
}

func runWalkForwardBacktest(ctx context.Context, engine *backtest.Engine, strat strategy.Strategy) {
	result, err := backtest.RunWalkForward(ctx, engine, strat, backtest.WalkForwardConfig{
		TrainingWindowDays:   90,
		ValidationWindowDays: 30,
		TestWindowDays:       30,
		StepSizeDays:         30,
		MinTradesPerWindow:   10,
	})
	if err != nil {
		engineLogger(engine).Fatalf("Walk-forward failed: %v", err)
	}
	engineLogger(engine).WithField("consistency", result.ConsistencyScore).Info("Walk-forward completed")
}

func runAllMethods(ctx context.Context, engine *backtest.Engine, cfg backtest.BacktestConfig, strat strategy.Strategy) {
	state, metrics, err := engine.Run(ctx, engineConfigStart(engine), engineConfigEnd(engine))
	if err != nil {
		engineLogger(engine).Fatalf("Historical backtest failed: %v", err)
	}
	probabilities := map[string]float64{}
	for _, bet := range state.Bets {
		probabilities[bet.ID.String()] = 0.5
	}
	monteCarlo, err := backtest.RunMonteCarlo(ctx, state.Bets, probabilities, backtest.MonteCarloConfig{
		Iterations:      cfg.MonteCarloIterations,
		CommissionRate:  cfg.CommissionRate,
		InitialBankroll: cfg.InitialBankroll,
	})
	if err != nil {
		engineLogger(engine).Fatalf("Monte Carlo failed: %v", err)
	}
	walkForward, err := backtest.RunWalkForward(ctx, engine, strat, backtest.WalkForwardConfig{
		TrainingWindowDays:   90,
		ValidationWindowDays: 30,
		TestWindowDays:       30,
		StepSizeDays:         30,
		MinTradesPerWindow:   10,
	})
	if err != nil {
		engineLogger(engine).Fatalf("Walk-forward failed: %v", err)
	}

	aggregated := backtest.AggregateResults(metrics, monteCarlo, walkForward, backtest.AggregationWeights{
		HistoricalReplay: 0.4,
		MonteCarlo:       0.3,
		WalkForward:      0.3,
	})
	report := backtest.GenerateConsoleReport(aggregated)
	engineLogger(engine).Info(report)

	if cfg.MLExportEnabled {
		export := backtest.MLExport{
			StrategyMetadata: strategy.StrategyMetadata{Name: strat.Name(), Parameters: strat.GetParameters()},
			BacktestSummary: backtest.BacktestSummary{
				StartDate:      cfg.StartDate,
				EndDate:        cfg.EndDate,
				InitialCapital: cfg.InitialBankroll,
				FinalCapital:   state.CurrentBankroll,
				TotalBets:      len(state.Bets),
			},
			Metrics: map[string]any{
				"historical":  metrics,
				"monte_carlo": monteCarlo,
				"walk_forward": walkForward,
			},
			BetHistory:        flattenBets(state.Bets),
			EquityCurve:       state.EquityCurve,
			ValidationResults: walkForward,
			RiskProfile: backtest.RiskProfile{
				VaR95:       monteCarlo.VaR95,
				VaR99:       monteCarlo.VaR99,
				MaxDrawdown: metrics.MaxDrawdown,
			},
			Recommendation: aggregated.Recommendation,
			CompositeScore: aggregated.CompositeScore,
			MLFeatures:     backtest.GenerateMLFeatures(aggregated),
		}
		if err := backtest.ExportToJSON(export, cfg.OutputPath); err != nil {
			engineLogger(engine).Fatalf("Failed to export ML JSON: %v", err)
		}

		params := backtest.ExportDBParams{
			StrategyID:     strategyIDFromName(strat.Name()),
			StartDate:      cfg.StartDate,
			EndDate:        cfg.EndDate,
			InitialCapital: cfg.InitialBankroll,
			FinalCapital:   state.CurrentBankroll,
		}
		if err := backtest.ExportToDatabase(ctx, aggregated, engine.Repositories().BacktestResult, params); err != nil {
			engineLogger(engine).Fatalf("Failed to persist backtest result: %v", err)
		}
	}
}

func flattenBets(bets []*models.Bet) []models.Bet {
	result := make([]models.Bet, 0, len(bets))
	for _, bet := range bets {
		if bet != nil {
			result = append(result, *bet)
		}
	}
	return result
}

func strategyIDFromName(name string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(name))
}

func engineConfigStart(engine *backtest.Engine) time.Time {
	return engine.Config().StartDate
}

func engineConfigEnd(engine *backtest.Engine) time.Time {
	return engine.Config().EndDate
}

func engineLogger(engine *backtest.Engine) *logrus.Logger {
	return engine.Logger()
}
