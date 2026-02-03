package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/database"
	applogger "github.com/yourusername/clever-better/internal/logger"
	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/service"
	"github.com/yourusername/clever-better/internal/tracing"
)

// Build information - set via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var (
	configFile string
	logger     *logrus.Logger
	mlLogger   *applogger.MLLogger
	cfg        *config.Config
	db         *database.DB
	repos      *repository.Repositories
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "./config/config.yaml", "Path to configuration file")
}

var rootCmd = &cobra.Command{
	Use:   "strategy-discovery",
	Short: "Discover and generate ML-driven betting strategies",
	Long:  `Executes the ML-driven strategy discovery pipeline to generate, evaluate, and activate new strategies.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := loadConfig(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if err := setupDependencies(); err != nil {
			return fmt.Errorf("failed to setup dependencies: %w", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		runDiscoveryPipeline()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func loadConfig() error {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CLEVER_BETTER")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	cfg = &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

func setupDependencies() error {
	// Setup logger
	logger = logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	mlLogger = applogger.NewMLLogger(logger)

	if os.Getenv("XRAY_ENABLED") == "true" {
		daemonAddr := os.Getenv("XRAY_DAEMON_ADDR")
		if daemonAddr == "" {
			daemonAddr = "localhost:2000"
		}
		tracing.Initialize(tracing.Config{
			ServiceName: "clever-better-strategy-discovery",
			Enabled:     true,
			SamplingRate: 0.1,
			DaemonAddr:  daemonAddr,
		}, logger)
		logger.WithField("daemon_addr", daemonAddr).Info("AWS X-Ray tracing initialized")
	}

	// Connect to database
	var err error
	db, err = database.New(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize repositories
	repos, err = repository.NewRepositories(db)
	if err != nil {
		return fmt.Errorf("failed to initialize repositories: %w", err)
	}

	return nil
}

func runDiscoveryPipeline() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create ML client
	mlClient, err := ml.NewCachedMLClient(&cfg.MLService, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create ML client")
	}
	defer mlClient.Close()

	// Create HTTP client
	httpClient := ml.NewHTTPClient(&cfg.MLService, logger)

	// Create services
	strategyGen := service.NewStrategyGeneratorService(mlClient, repos.Strategy, repos.BacktestResult, logger)
	mlFeedback := service.NewMLFeedbackService(mlClient, httpClient, repos.BacktestResult, logger)
	strategyEval := service.NewStrategyEvaluatorService(mlClient, repos.Strategy, repos.BacktestResult, logger)
	orchestrator := service.NewMLOrchestratorService(strategyGen, mlFeedback, strategyEval, mlClient, repos.Prediction, logger)

	// Configuration for discovery pipeline
	discoveryConfig := service.DiscoveryConfig{
		GenerateCount:       10,
		RiskLevel:           "medium",
		TargetReturn:        0.15,
		MinCompositeScore:   0.65,
		DeactivateThreshold: 0.50,
		SubmitFeedback:      true,
		TriggerRetraining:   true,
	}

	// Run discovery pipeline
	logger.Info("Starting strategy discovery pipeline")
	mlLogger.LogStrategyGeneration(map[string]interface{}{"risk_level": discoveryConfig.RiskLevel}, discoveryConfig.GenerateCount, 0, 0)
	report, err := orchestrator.RunStrategyDiscoveryPipeline(ctx, discoveryConfig)
	if err != nil {
		logger.WithError(err).Error("Pipeline execution failed")
		mlLogger.LogMLPredictionError("strategy_discovery", err.Error())
		os.Exit(1)
	}
	if len(report.TopStrategies) > 0 {
		mlLogger.LogStrategyRankingUpdate(report.GeneratedCount, report.TopStrategies[0].StrategyID, "composite_score")
	}

	// Print report
	fmt.Println("\n=== Strategy Discovery Pipeline Report ===")
	fmt.Printf("Run ID: %s\n", report.RunID)
	fmt.Printf("Generated Strategies: %d\n", report.GeneratedCount)
	fmt.Printf("Activated Strategies: %d\n", report.ActivatedCount)
	fmt.Printf("Deactivated Strategies: %d\n", report.DeactivatedCount)
	fmt.Printf("Feedback Submitted: %d\n", report.FeedbackSubmitted)
	fmt.Printf("Retraining Triggered: %v\n", report.RetrainingTriggered)
	fmt.Printf("Duration: %v\n", report.Duration)
	fmt.Printf("\nTop Strategies:\n")
	for i, strategy := range report.TopStrategies {
		fmt.Printf("  %d. %s (Score: %.2f, Rank: %d)\n", i+1, strategy.StrategyName, strategy.CompositeScore, strategy.Rank)
	}
	fmt.Printf("\nCompleted at: %s\n", report.CompletedAt)
}
