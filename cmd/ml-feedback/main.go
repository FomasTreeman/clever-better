package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/yourusername/clever-better/internal/config"
	applogger "github.com/yourusername/clever-better/internal/logger"
	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/service"
	"github.com/yourusername/clever-better/internal/tracing"
	"github.com/yourusername/clever-better/internal/database"
)

// Build information - set via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var (
	configFile  string
	batchSize   int
	logger      *logrus.Logger
	mlLogger    *applogger.MLLogger
	cfg         *config.Config
	db          *database.DB
	repos       *repository.Repositories
	mlFeedback  *service.MLFeedbackService
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "./config/config.yaml", "Path to configuration file")
	submitCmd.Flags().IntVarP(&batchSize, "batch-size", "b", 100, "Number of backtest results to submit per batch")
}

var rootCmd = &cobra.Command{
	Use:   "ml-feedback",
	Short: "Submit backtest feedback to ML service",
	Long:  `Submit backtest results as feedback to train and improve ML models.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := loadConfig(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if err := setupDependencies(); err != nil {
			return fmt.Errorf("failed to setup dependencies: %w", err)
		}
		return nil
	},
}

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit batch feedback",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return submitBatchFeedback(ctx)
	},
}

var retrainCmd = &cobra.Command{
	Use:   "retrain",
	Short: "Trigger model retraining",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return triggerRetraining(ctx)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check ML service health",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return checkMLServiceHealth(ctx)
	},
}

func main() {
	rootCmd.AddCommand(submitCmd, retrainCmd, statusCmd)

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

	// Initialize ML logger
	mlLogger = applogger.NewMLLogger(logger)

	// Initialize X-Ray tracing
	if os.Getenv("XRAY_ENABLED") == "true" {
		daemonAddr := os.Getenv("XRAY_DAEMON_ADDR")
		if daemonAddr == "" {
			daemonAddr = "localhost:2000"
		}
		tracing.Initialize(tracing.Config{
			ServiceName: "clever-better-ml-feedback",
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

	// Create ML clients
	mlClient, err := ml.NewCachedMLClient(&cfg.MLService, logger)
	if err != nil {
		return fmt.Errorf("failed to create ML client: %w", err)
	}

	httpClient := ml.NewHTTPClient(&cfg.MLService, logger)
	mlFeedback = service.NewMLFeedbackService(mlClient, httpClient, repos.BacktestResult, logger)

	return nil
}

func submitBatchFeedback(ctx context.Context) error {
	logger.WithField("batch_size", batchSize).Info("Submitting batch feedback")

	count, err := mlFeedback.SubmitBatch(ctx, batchSize)
	if err != nil {
		logger.WithError(err).Error("Failed to submit batch feedback")
		mlLogger.LogMLPredictionError("feedback_submission", err.Error())
		return err
	}
	mlLogger.LogBacktestFeedback("batch", count, 0)

	fmt.Printf("Successfully submitted %d backtest results as feedback\n", count)
	return nil
}

func triggerRetraining(ctx context.Context) error {
	logger.Info("Triggering model retraining")

	configs := []ml.TrainingConfig{
		{
			ModelType:            "classifier",
			Epochs:               50,
			BatchSize:            32,
			LearningRate:         0.001,
			HyperparameterSearch: true,
		},
		{
			ModelType:            "ensemble",
			Epochs:               30,
			BatchSize:            64,
			LearningRate:         0.01,
			HyperparameterSearch: false,
		},
		{
			ModelType:            "rl_agent",
			Epochs:               100,
			BatchSize:            128,
			LearningRate:         0.0001,
			HyperparameterSearch: false,
		},
	}

	for _, config := range configs {
		status, err := mlFeedback.TriggerRetraining(ctx, config)
		if err != nil {
			logger.WithError(err).WithField("model_type", config.ModelType).Error("Failed to trigger retraining")
			mlLogger.LogMLPredictionError(config.ModelType, err.Error())
			continue
		}
		mlLogger.LogModelTraining(config.ModelType, 0, map[string]float64{}, map[string]interface{}{"batch_size": config.BatchSize, "epochs": config.Epochs})

		fmt.Printf("✓ Submitted training job for %s model\n", config.ModelType)
		fmt.Printf("  Job ID: %s\n", status.JobID)
		fmt.Printf("  Status: %s\n", status.Status)
	}

	return nil
}

func checkMLServiceHealth(ctx context.Context) error {
	httpClient := ml.NewHTTPClient(&cfg.MLService, logger)
	
	if err := httpClient.HealthCheck(ctx); err != nil {
		fmt.Printf("❌ ML service is unavailable: %v\n", err)
		return err
	}

	fmt.Println("✓ ML service is healthy")
	return nil
}
