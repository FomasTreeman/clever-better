// Package config provides configuration management for the Clever Better application.
package config

import (
	"fmt"
)

// Config represents the complete application configuration
type Config struct {
	App            AppConfig            `mapstructure:"app" validate:"required"`
	Database       DatabaseConfig       `mapstructure:"database" validate:"required"`
	Betfair        BetfairConfig        `mapstructure:"betfair" validate:"required"`
	MLService      MLServiceConfig      `mapstructure:"ml_service" validate:"required"`
	Trading        TradingConfig        `mapstructure:"trading" validate:"required"`
	Backtest       BacktestConfig       `mapstructure:"backtest" validate:"required"`
	DataIngestion  DataIngestionConfig  `mapstructure:"data_ingestion" validate:"required"`
	Metrics        MetricsConfig        `mapstructure:"metrics" validate:"required"`
	Features       FeaturesConfig       `mapstructure:"features" validate:"required"`
	Bot            BotConfig            `mapstructure:"bot" validate:"required"`
}

// AppConfig represents application-level configuration
type AppConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required,environment"`
	LogLevel    string `mapstructure:"log_level" validate:"required,loglevel"`
}

// DatabaseConfig represents database connection configuration
type DatabaseConfig struct {
	Host                string `mapstructure:"host" validate:"required"`
	Port                int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	Name                string `mapstructure:"name" validate:"required"`
	User                string `mapstructure:"user" validate:"required"`
	Password            string `mapstructure:"password" validate:"required"`
	SSLMode             string `mapstructure:"ssl_mode" validate:"required,oneof=disable require verify-full"`
	MaxConnections      int    `mapstructure:"max_connections" validate:"required,gt=0"`
	MaxIdleConnections  int    `mapstructure:"max_idle_connections" validate:"required,gt=0"`
}

// BetfairConfig represents Betfair API configuration
type BetfairConfig struct {
	APIURL    string `mapstructure:"api_url" validate:"required,url"`
	StreamURL string `mapstructure:"stream_url" validate:"required"`
	AppKey    string `mapstructure:"app_key" validate:"required"`
	Username  string `mapstructure:"username" validate:"required"`
	Password  string `mapstructure:"password" validate:"required"`
	CertFile  string `mapstructure:"cert_file" validate:"required"`
	KeyFile   string `mapstructure:"key_file" validate:"required"`
}

// MLServiceConfig represents ML service configuration
type MLServiceConfig struct {
	URL                    string `mapstructure:"url" validate:"required,url"`
	HTTPAddress            string `mapstructure:"http_address" validate:"required"`
	GRPCAddress            string `mapstructure:"grpc_address" validate:"required"`
	TimeoutSeconds         int    `mapstructure:"timeout_seconds" validate:"required,gt=0"`
	RequestTimeoutSeconds  int    `mapstructure:"request_timeout_seconds" validate:"required,gt=0"`
	RetryAttempts          int    `mapstructure:"retry_attempts" validate:"required,gte=0"`
	CacheTTLSeconds        int    `mapstructure:"cache_ttl_seconds" validate:"required,gt=0"`
	CacheMaxSize           int    `mapstructure:"cache_max_size" validate:"required,gt=0"`
	EnableStrategyGeneration bool `mapstructure:"enable_strategy_generation"`
	EnableFeedbackLoop     bool   `mapstructure:"enable_feedback_loop"`
	FeedbackBatchSize      int    `mapstructure:"feedback_batch_size" validate:"required,gt=0"`
	RetrainingIntervalHours int  `mapstructure:"retraining_interval_hours" validate:"required,gt=0"`
}

// TradingConfig represents trading strategy and risk management configuration
type TradingConfig struct {
	MaxStakePerBet               float64  `mapstructure:"max_stake_per_bet" validate:"required,gt=0"`
	MaxDailyLoss                 float64  `mapstructure:"max_daily_loss" validate:"required,gt=0"`
	MaxExposure                  float64  `mapstructure:"max_exposure" validate:"required,gt=0"`
	MinConfidenceThreshold       float64  `mapstructure:"min_confidence_threshold" validate:"required,gte=0,lte=1"`
	MinExpectedValue             float64  `mapstructure:"min_expected_value" validate:"required,gte=0"`
	Markets                      []string `mapstructure:"markets" validate:"required,min=1,markets"`
	PreRaceWindowMinutes         int      `mapstructure:"pre_race_window_minutes" validate:"required,gte=0"`
	MinTimeToStartSeconds        int      `mapstructure:"min_time_to_start_seconds" validate:"required,gte=0"`
	MaxConcurrentBets            int      `mapstructure:"max_concurrent_bets" validate:"required,gt=0"`
	StrategyEvaluationInterval   int      `mapstructure:"strategy_evaluation_interval" validate:"required,gt=0"`
	EmergencyShutdownEnabled     bool     `mapstructure:"emergency_shutdown_enabled"`
}

// BotConfig represents bot-specific configuration
type BotConfig struct {
	OrderMonitoringInterval    int     `mapstructure:"order_monitoring_interval" validate:"required,gt=0"`
	PerformanceUpdateInterval  int     `mapstructure:"performance_update_interval" validate:"required,gt=0"`
	MaxConsecutiveLosses       int     `mapstructure:"max_consecutive_losses" validate:"required,gt=0"`
	MaxDrawdownPercent         float64 `mapstructure:"max_drawdown_percent" validate:"required,gt=0,lt=1"`
	RiskFreeRate               float64 `mapstructure:"risk_free_rate" validate:"gte=0,lte=1"`
}

// BacktestConfig represents backtesting configuration
type BacktestConfig struct {
	StartDate             string  `mapstructure:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate               string  `mapstructure:"end_date" validate:"required,datetime=2006-01-02"`
	InitialBankroll       float64 `mapstructure:"initial_bankroll" validate:"required,gt=0"`
	MonteCarloIterations  int     `mapstructure:"monte_carlo_iterations" validate:"required,gt=0"`
	WalkForwardWindows    int     `mapstructure:"walk_forward_windows" validate:"required,gt=0"`
	CommissionRate        float64 `mapstructure:"commission_rate" validate:"required,gte=0,lte=0.1"`
	SlippageTicks         int     `mapstructure:"slippage_ticks" validate:"required,gte=0"`
	MinLiquidity          float64 `mapstructure:"min_liquidity" validate:"required,gte=0"`
	OutputPath            string  `mapstructure:"output_path" validate:"required"`
	MLExportEnabled       bool    `mapstructure:"ml_export_enabled"`
	RiskFreeRate          float64 `mapstructure:"risk_free_rate" validate:"gte=0"`
}

// DataIngestionConfig represents data ingestion configuration
type DataIngestionConfig struct {
	Sources  []DataSourceConfig `mapstructure:"sources" validate:"required,min=1"`
	Schedule ScheduleConfig     `mapstructure:"schedule" validate:"required"`
}

// DataSourceConfig represents a single data source configuration
type DataSourceConfig struct {
	Name      string `mapstructure:"name" validate:"required"`
	Enabled   bool   `mapstructure:"enabled"`
	BatchSize int    `mapstructure:"batch_size" validate:"omitempty,gt=0"`
	APIKey    string `mapstructure:"api_key"`
}

// ScheduleConfig represents data ingestion scheduling
type ScheduleConfig struct {
	HistoricalSync              string `mapstructure:"historical_sync" validate:"required"`
	LivePollingIntervalSeconds  int    `mapstructure:"live_polling_interval_seconds" validate:"required,gt=0"`
}

// MetricsConfig represents metrics and monitoring configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	Path    string `mapstructure:"path" validate:"required"`
}

// FeaturesConfig represents feature flags
type FeaturesConfig struct {
	LiveTradingEnabled      bool `mapstructure:"live_trading_enabled"`
	PaperTradingEnabled     bool `mapstructure:"paper_trading_enabled"`
	MLPredictionsEnabled    bool `mapstructure:"ml_predictions_enabled"`
	AdvancedAnalyticsEnabled bool `mapstructure:"advanced_analytics_enabled"`
}

// IsDevelopment checks if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsStaging checks if the application is running in staging mode
func (c *Config) IsStaging() bool {
	return c.App.Environment == "staging"
}

// IsProduction checks if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// GetDatabaseDSN returns a PostgreSQL DSN string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetMLServiceHTTPURL returns the formatted HTTP URL for ML service
func (c *Config) GetMLServiceHTTPURL() string {
	return c.MLService.URL
}

// GetMLServiceGRPCAddress returns the gRPC address for ML service
func (c *Config) GetMLServiceGRPCAddress() string {
	return c.MLService.GRPCAddress
}
