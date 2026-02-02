// Package main provides the entry point for the data ingestion service.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/datasource"
	dbpkg "github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/logger"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/scheduler"
	"github.com/yourusername/clever-better/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config/config.yaml")
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

	// Set up logging
	log := logger.NewLogger(cfg.App.LogLevel)

	log.Info("Clever Better Data Ingestion Service")
	log.Infof("Running in %s mode with log level: %s", cfg.App.Environment, cfg.App.LogLevel)
	log.Info("Configuration loaded and validated successfully")

	// Initialize database connection
	db, err := dbpkg.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Info("Database connection established")

	// Initialize repositories
	repos, err := repository.NewRepositories(db)
	if err != nil {
		log.Fatalf("Failed to create repositories: %v", err)
	}

	// Initialize HTTP client
	httpClientCfg := datasource.DefaultHTTPClientConfig()
	httpClientCfg.RateLimit = float64(cfg.App.RateLimit.RequestsPerSecond)
	httpClient := datasource.NewRateLimitedHTTPClient(httpClientCfg, log)
	defer httpClient.Close()

	// Initialize data source factory
	factory := datasource.NewFactory(cfg, log)

	// Create data sources
	sources, err := factory.NewDataSources(cfg.DataIngestion, httpClient)
	if err != nil {
		log.Fatalf("Failed to create data sources: %v", err)
	}

	if len(sources) == 0 {
		log.Fatalf("No data sources configured")
	}

	// Create data validator
	validator := datasource.NewDataValidator(log)

	// Create data normalizer
	normalizer := datasource.NewDataNormalizer(log)

	// Initialize ingestion service
	ingestionSvc := service.NewIngestionService(
		sources,
		repos.Race,
		repos.Runner,
		validator,
		normalizer,
		log,
		100, // batch size
	)

	log.Info("Ingestion service initialized")

	// Initialize scheduler
	sched := scheduler.NewScheduler(ingestionSvc, log)

	// Schedule jobs based on configuration
	if cfg.App.Scheduler.HistoricalSyncEnabled {
		if err := sched.ScheduleHistoricalSync(
			cfg.App.Scheduler.HistoricalSyncCronExpression,
			"betfair_historical",
		); err != nil {
			log.Warnf("Failed to schedule historical sync: %v", err)
		}
	}

	if cfg.App.Scheduler.LivePollingEnabled {
		if err := sched.ScheduleLivePolling(
			cfg.App.Scheduler.LivePollingIntervalSeconds,
			"betfair_historical",
		); err != nil {
			log.Warnf("Failed to schedule live polling: %v", err)
		}
	}

	// Start scheduler
	if err := sched.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	log.Info("Scheduler started")

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Infof("Received signal: %v", sig)

		// Gracefully stop scheduler with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := sched.Stop(); err != nil {
			log.Errorf("Error stopping scheduler: %v", err)
		}

		log.Info("Graceful shutdown complete")
		os.Exit(0)
	}()

	// Keep the service running
	select {}
}
