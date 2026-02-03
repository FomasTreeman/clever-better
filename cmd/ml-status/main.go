package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/service"
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
	cfg        *config.Config
	db         *database.DB
	repos      *repository.Repositories
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "./config/config.yaml", "Path to configuration file")
}

var rootCmd = &cobra.Command{
	Use:   "ml-status",
	Short: "Check ML service and pipeline status",
	Long:  `Displays health status and metrics for the ML service and ML integration pipeline.`,
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
		displayStatus()
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
	logger.SetLevel(logrus.WarnLevel)

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

func displayStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ML Service Integration Status                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")

	// Check ML service health
	fmt.Print("ML Service Health: ")
	httpClient := ml.NewHTTPClient(&cfg.MLService, logger)
	if err := httpClient.HealthCheck(ctx); err != nil {
		fmt.Println("❌ UNAVAILABLE")
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Println("✓ ONLINE")
	}

	// Get ML client to check cache stats
	mlClient, err := ml.NewCachedMLClient(&cfg.MLService, logger)
	if err == nil {
		defer mlClient.Close()
		hits, misses, ratio := mlClient.GetCacheStats()
		fmt.Printf("\nCache Statistics:\n")
		fmt.Printf("  Hits: %d\n", hits)
		fmt.Printf("  Misses: %d\n", misses)
		fmt.Printf("  Hit Ratio: %.2f%%\n", ratio*100)
	}

	// Get database statistics
	fmt.Println("\nDatabase Statistics:")
	displayDatabaseStats(ctx)

	// Configuration
	fmt.Println("\nConfiguration:")
	fmt.Printf("  ML Service URL: %s\n", cfg.MLService.URL)
	fmt.Printf("  gRPC Address: %s\n", cfg.MLService.GRPCAddress)
	fmt.Printf("  Cache TTL: %d seconds\n", cfg.MLService.CacheTTLSeconds)
	fmt.Printf("  Cache Max Size: %d\n", cfg.MLService.CacheMaxSize)
	fmt.Printf("  Strategy Generation: %v\n", cfg.MLService.EnableStrategyGeneration)
	fmt.Printf("  Feedback Loop: %v\n", cfg.MLService.EnableFeedbackLoop)
	fmt.Printf("  Retraining Interval: %d hours\n", cfg.MLService.RetrainingIntervalHours)

	fmt.Println("\n")
}

func displayDatabaseStats(ctx context.Context) {
	// This is a placeholder - in a real implementation, you would query actual statistics
	fmt.Println("  Backtest Results: [retrieving...]")
	fmt.Println("  Active Strategies: [retrieving...]")
	fmt.Println("  Recent Predictions: [retrieving...]")
}
