// Package main provides the entry point for the trading bot.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/betfair"
	"github.com/yourusername/clever-better/internal/bot"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/datasource"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/logger"
	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/repository"
)

func main() {
	var (
		cfg    *config.Config
		err    error
		appLog *logrus.Logger
		db     *database.DB
	)

	// Load configuration
	cfg, err = config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load AWS secrets if enabled
	if os.Getenv("AWS_SECRETS_ENABLED") == "true" {
		region := os.Getenv("AWS_REGION")
		secretName := os.Getenv("AWS_SECRET_NAME")
		if region == "" || secretName == "" {
			log.Fatalf("AWS_REGION and AWS_SECRET_NAME environment variables must be set when AWS_SECRETS_ENABLED is true")
		}
		if err := config.LoadSecretsFromAWS(cfg, region, secretName); err != nil {
			log.Fatalf("Failed to load secrets: %v", err)
		}
	}

	// Validate configuration
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	if !cfg.Features.LiveTradingEnabled && !cfg.Features.PaperTradingEnabled {
		log.Fatalf("At least one trading mode must be enabled")
	}

	// Set up logging
	appLog = logger.NewLogger(cfg.App.LogLevel)
	appLog.WithFields(logrus.Fields{
		"environment": cfg.App.Environment,
		"log_level":   cfg.App.LogLevel,
	}).Info("Clever Better Trading Bot starting")

	// Initialize database connection
	db, err = database.NewDB(cfg.GetDatabaseDSN())
	if err != nil {
		appLog.WithError(err).Fatal("Failed to connect to database")
	}
	defer func() {
		if err := db.Close(context.Background()); err != nil {
			appLog.WithError(err).Error("Failed to close database connection")
		}
	}()

	appLog.Info("Database connection established")

	// Run migrations if needed
	// Note: In production, migrations should be run separately
	// db.RunMigrations()

	// Initialize repositories
	raceRepo := repository.NewPostgresRaceRepository(db)
	runnerRepo := repository.NewPostgresRunnerRepository(db)
	oddsRepo := repository.NewPostgresOddsRepository(db)
	betRepo := repository.NewPostgresBetRepository(db)
	strategyRepo := repository.NewPostgresStrategyRepository(db)
	strategyPerfRepo := repository.NewPostgresStrategyPerformanceRepository(db)

	// Initialize ML client
	mlClient := ml.NewMLClient(&cfg.MLService, appLog)
	cachedMLClient := ml.NewCachedMLClient(mlClient, appLog)

	appLog.WithField("ml_service_url", cfg.MLService.URL).Info("ML client initialized")

	var (
		bettingService *betfair.BettingService
		orderManager   *betfair.OrderManager
	)

	if cfg.Features.LiveTradingEnabled {
		orderLogger := log.New(os.Stdout, "order-manager: ", log.LstdFlags)
		httpLogger := log.New(os.Stdout, "betfair-http: ", log.LstdFlags)
		httpClient := datasource.NewRateLimitedHTTPClient(datasource.DefaultHTTPClientConfig(), httpLogger)

		// Initialize Betfair client
		betfairClient := betfair.NewBetfairClient(&cfg.Betfair, httpClient, orderLogger)

		// Login to Betfair
		if err := betfairClient.Login(context.Background()); err != nil {
			appLog.WithError(err).Fatal("Failed to login to Betfair")
		}
		defer func() {
			if err := betfairClient.Logout(context.Background()); err != nil {
				appLog.WithError(err).Error("Failed to logout from Betfair")
			}
		}()

		appLog.Info("Betfair client initialized and logged in")

		// Initialize betting service
		bettingService = betfair.NewBettingService(
			betfairClient,
			betRepo,
			betfair.BettingConfig{
				MaxStake:       cfg.Trading.MaxStakePerBet,
				MinStake:       0.10,
				MaxBetsPerDay:  cfg.Trading.MaxConcurrentBets,
				CommissionRate: cfg.Backtest.CommissionRate,
			},
			orderLogger,
		)

		// Initialize order manager
		orderManager = betfair.NewOrderManager(
			bettingService,
			betRepo,
			time.Duration(cfg.Bot.OrderMonitoringInterval)*time.Second,
			orderLogger,
		)
	} else {
		appLog.Info("Live trading disabled; skipping Betfair initialization")
	}

	// Create bot orchestrator
	repos := bot.Repositories{
		Strategy:            strategyRepo,
		Race:                raceRepo,
		Runner:              runnerRepo,
		Odds:                oddsRepo,
		Bet:                 betRepo,
		StrategyPerformance: strategyPerfRepo,
	}

	orchestrator, err := bot.NewOrchestrator(
		cfg,
		db,
		cachedMLClient,
		bettingService,
		orderManager,
		repos,
		appLog,
	)
	if err != nil {
		appLog.WithError(err).Fatal("Failed to create orchestrator")
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start orchestrator
	if err := orchestrator.Start(ctx); err != nil {
		appLog.WithError(err).Fatal("Failed to start orchestrator")
	}

	appLog.WithFields(logrus.Fields{
		"paper_trading":         cfg.Features.PaperTradingEnabled,
		"live_trading":          cfg.Features.LiveTradingEnabled,
		"ml_predictions":        cfg.Features.MLPredictionsEnabled,
		"emergency_shutdown":    cfg.Trading.EmergencyShutdownEnabled,
	}).Info("Bot orchestrator started successfully")

	// Print status information
	status := orchestrator.GetStatus()
	appLog.WithFields(logrus.Fields{
		"active_strategies":     status.ActiveStrategies,
		"circuit_breaker_state": status.CircuitBreakerState,
		"max_exposure":          status.RiskMetrics.MaxExposure,
		"max_daily_loss":        status.RiskMetrics.MaxDailyLoss,
	}).Info("Bot is running")

	// Wait for shutdown signal
	sig := <-sigChan
	appLog.WithField("signal", sig).Info("Shutdown signal received")

	// Graceful shutdown
	appLog.Info("Initiating graceful shutdown...")

	// Cancel context to stop all goroutines
	cancel()

	// Stop orchestrator
	if err := orchestrator.Stop(); err != nil {
		appLog.WithError(err).Error("Error during orchestrator shutdown")
	}

	// Give components time to cleanup
	time.Sleep(2 * time.Second)

	appLog.Info("Clever Better Trading Bot shut down successfully")
}
