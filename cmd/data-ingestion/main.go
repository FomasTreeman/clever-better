// Package main provides the entry point for the data ingestion service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/datasource"
	dbpkg "github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/health"
	"github.com/yourusername/clever-better/internal/logger"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/scheduler"
	"github.com/yourusername/clever-better/internal/service"
)

// Build information - set via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// initConfig loads and validates configuration for data ingestion service
func initConfig() (*config.Config, error) {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load AWS secrets if enabled
	if os.Getenv("AWS_SECRETS_ENABLED") == "true" {
		region := os.Getenv("AWS_REGION")
		secretName := os.Getenv("AWS_SECRET_NAME")
		if region == "" || secretName == "" {
			return nil, fmt.Errorf("AWS_REGION and AWS_SECRET_NAME environment variables must be set when AWS_SECRETS_ENABLED is true")
		}
		if err := config.LoadSecretsFromAWS(cfg, region, secretName); err != nil {
			return nil, fmt.Errorf("failed to load secrets: %w", err)
		}
	}

	// Validate configuration
	if err := config.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// createDataSources initializes and validates data sources from configuration
func createDataSources(cfg *config.Config, httpClient datasource.HTTPClient, appLog logger.Interface) ([]datasource.DataSource, error) {
	factory := datasource.NewFactory(cfg, appLog)
	sources, err := factory.NewDataSources(cfg.DataIngestion, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create data sources: %w", err)
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no data sources configured")
	}

	return sources, nil
}

// scheduleJobs configures and schedules data ingestion jobs
func scheduleJobs(cfg *config.Config, sched *scheduler.Scheduler, appLog logger.Interface) error {
	if cfg.App.Scheduler.HistoricalSyncEnabled {
		if err := sched.ScheduleHistoricalSync(
			cfg.App.Scheduler.HistoricalSyncCronExpression,
			"betfair_historical",
		); err != nil {
			return fmt.Errorf("failed to schedule historical sync: %w", err)
		}
	}

	if cfg.App.Scheduler.LivePollingEnabled {
		if err := sched.ScheduleLivePolling(
			cfg.App.Scheduler.LivePollingIntervalSeconds,
			"betfair_historical",
		); err != nil {
			return fmt.Errorf("failed to schedule live polling: %w", err)
		}
	}

	return nil
}

// handleGracefulShutdown manages the shutdown sequence
func handleGracefulShutdown(sigChan chan os.Signal, cancel context.CancelFunc, sched *scheduler.Scheduler, healthServer *health.Server, appLog logger.Interface) {
	sig := <-sigChan
	appLog.Infof("Received signal: %v", sig)

	// Mark as not ready
	healthServer.SetReady(false)

	// Cancel main context
	cancel()

	// Gracefully stop scheduler with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := sched.Stop(); err != nil {
		appLog.Errorf("Error stopping scheduler: %v", err)
	}

	_ = shutdownCtx // Satisfy unused variable if needed

	appLog.Info("Graceful shutdown complete")
	os.Exit(0)
}

func main() {
	// Handle version flag
	versionFlag := flag.Bool("version", false, "Print version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Clever Better Data Ingestion Service\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// Load and validate configuration
	cfg, err := initConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Set up logging
	appLog := logger.NewLogger(cfg.App.LogLevel)
	appLog.Info("Clever Better Data Ingestion Service")
	appLog.Infof("Version: %s, Commit: %s, Build Date: %s", Version, GitCommit, BuildDate)
	appLog.Infof("Running in %s mode with log level: %s", cfg.App.Environment, cfg.App.LogLevel)
	appLog.Info("Configuration loaded and validated successfully")

	// Initialize database connection
	db, err := dbpkg.NewConnection(cfg.Database)
	if err != nil {
		appLog.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	appLog.Info("Database connection established")

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health check server
	healthServer := health.NewServer(health.Config{
		ServiceName: "data-ingestion",
		Version:     Version,
		Commit:      GitCommit,
		Logger:      appLog,
		DB:          nil, // Connection interface differs; health server will skip DB check
	})

	if err := healthServer.Start(ctx); err != nil {
		appLog.Errorf("Failed to start health server: %v", err)
	} else {
		appLog.Info("Health check server started")
	}
	defer healthServer.Shutdown()

	// Initialize repositories
	repos, err := repository.NewRepositories(db)
	if err != nil {
		appLog.Fatalf("Failed to create repositories: %v", err)
	}

	// Initialize HTTP client
	httpClientCfg := datasource.DefaultHTTPClientConfig()
	httpClientCfg.RateLimit = float64(cfg.App.RateLimit.RequestsPerSecond)
	httpClient := datasource.NewRateLimitedHTTPClient(httpClientCfg, appLog)
	defer httpClient.Close()

	// Create data sources
	sources, err := createDataSources(cfg, httpClient, appLog)
	if err != nil {
		appLog.Fatalf("Data source initialization error: %v", err)
	}

	// Create data validator and normalizer
	validator := datasource.NewDataValidator(appLog)
	normalizer := datasource.NewDataNormalizer(appLog)

	// Initialize ingestion service
	ingestionSvc := service.NewIngestionService(
		sources,
		repos.Race,
		repos.Runner,
		validator,
		normalizer,
		appLog,
		100, // batch size
	)

	appLog.Info("Ingestion service initialized")

	// Initialize scheduler
	sched := scheduler.NewScheduler(ingestionSvc, appLog)

	// Schedule jobs based on configuration
	if err := scheduleJobs(cfg, sched, appLog); err != nil {
		appLog.Warnf("Job scheduling error: %v", err)
	}

	// Start scheduler
	if err := sched.Start(); err != nil {
		appLog.Fatalf("Failed to start scheduler: %v", err)
	}

	appLog.Info("Scheduler started")

	// Mark health server as ready
	healthServer.SetReady(true)

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go handleGracefulShutdown(sigChan, cancel, sched, healthServer, appLog)

	// Keep the service running
	select {}
}
