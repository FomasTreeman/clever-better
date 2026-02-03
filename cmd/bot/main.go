// Package main provides the entry point for the trading bot.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/betfair"
	"github.com/yourusername/clever-better/internal/bot"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/datasource"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/health"
	"github.com/yourusername/clever-better/internal/logger"
	"github.com/yourusername/clever-better/internal/metrics"
	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/tracing"
)

// Build information - set via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Handle version flag
	versionFlag := flag.Bool("version", false, "Print version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Clever Better Trading Bot\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		os.Exit(0)
	}
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
		"version":     Version,
		"commit":      GitCommit,
		"build_date":  BuildDate,
	}).Info("Clever Better Trading Bot starting")

	// Initialize Prometheus metrics registry
	metrics.InitRegistry()
	appLog.Info("Prometheus metrics registry initialized")

	// Start Prometheus HTTP server on port 9090
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(
			metrics.GetRegistry(),
			promhttp.HandlerOpts{},
		))
		metricsServer := &http.Server{
			Addr:         ":9090",
			Handler:      http.DefaultServeMux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		}
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLog.WithError(err).Error("Prometheus metrics server error")
		}
	}()
	appLog.Info("Prometheus metrics server started on :9090")

	// Initialize AWS X-Ray tracing
	xrayEnabled := os.Getenv("XRAY_ENABLED") == "true"
	if xrayEnabled {
		daemonAddr := os.Getenv("XRAY_DAEMON_ADDR")
		if daemonAddr == "" {
			daemonAddr = "localhost:2000"
		}
		tracing.Initialize(tracing.Config{
			ServiceName: "clever-better-bot",
			Enabled:     true,
			SamplingRate: 0.1,
			DaemonAddr:  daemonAddr,
		}, appLog)
		appLog.WithField("daemon_addr", daemonAddr).Info("AWS X-Ray tracing initialized")
	}
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

	// Set up signal handling and context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create specialized loggers for observability
	strategyLogger := logger.NewStrategyLogger(appLog)
	mlLogger := logger.NewMLLogger(appLog)
	auditLogger := logger.NewAuditLogger(appLog)

	// Start health check server
	healthServer := health.NewServer(health.Config{
		ServiceName: "bot",
		Version:     Version,
		Commit:      GitCommit,
		Logger:      appLog,
		DB:          db,
	})

	if err := healthServer.Start(ctx); err != nil {
		appLog.WithError(err).Error("Failed to start health server")
	} else {
		appLog.Info("Health check server started")
	}
	defer healthServer.Shutdown()

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
		strategyLogger.Entry,
		mlLogger.Entry,
		auditLogger.Entry,
	)
	if err != nil {
		appLog.WithError(err).Fatal("Failed to create orchestrator")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Mark health server as ready
	healthServer.SetReady(true)

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
