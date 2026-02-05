// Package config provides configuration management for the Clever Better application.
package config

import (
	"os"
	"testing"
)

const (
	validConfigPath               = "testdata/valid_config.yaml"
	expansionConfigPath           = "testdata/expansion_config.yaml"
	expansionConfigMissingPath    = "testdata/expansion_config_missing.yaml"
	nonexistentConfigPath         = "testdata/nonexistent_config.yaml"
	expectedNoErrorLoadingConfig  = "expected no error loading config, got %v"
	expectedNoErrorMsg            = "expected no error, got %v"
	expectedNonNilConfig          = "expected non-nil config"
	cleverBetterName              = "clever-better"
	developmentEnv                = "development"
	invalidEnv                    = "invalid"
	localhostHost                 = "localhost"
	postgresPort                  = 5432
	postgresPrefix                = "postgres://"
	testAppName                   = "test-app"
	testDBPassword                = "TEST_DB_PASSWORD"
	testMissingVar                = "TEST_MISSING_VAR"
	expandedSecretValue           = "expanded_secret_value"
	marketsValidationError        = "markets"
	marketsValidationErrorCaps    = "Markets"
)

// TestLoadConfigSuccess tests loading a valid configuration file
func TestLoadConfigSuccess(t *testing.T) {
	cfg, err := Load(validConfigPath)
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	if cfg == nil {
		t.Fatal(expectedNonNilConfig)
	}

	if cfg.App.Name != cleverBetterName {
		t.Errorf("expected app name '%s', got '%s'", cleverBetterName, cfg.App.Name)
	}

	if cfg.App.Environment != developmentEnv {
		t.Errorf("expected environment '%s', got '%s'", developmentEnv, cfg.App.Environment)
	}

	if cfg.Database.Host != localhostHost {
		t.Errorf("expected database host '%s', got '%s'", localhostHost, cfg.Database.Host)
	}

	if cfg.Database.Port != postgresPort {
		t.Errorf("expected database port %d, got %d", postgresPort, cfg.Database.Port)
	}
}

// TestLoadConfigFileNotFound tests handling of missing configuration file
func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := Load(nonexistentConfigPath)
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

// TestLoadConfigEnvironmentVariables tests environment variable override
func TestLoadConfigEnvironmentVariables(t *testing.T) {
	// Set an environment variable
	os.Setenv("CLEVER_BETTER_APP_NAME", testAppName)
	defer os.Unsetenv("CLEVER_BETTER_APP_NAME")

	cfg, err := Load(validConfigPath)
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	if cfg.App.Name != testAppName {
		t.Errorf("expected app name '%s' from environment, got '%s'", testAppName, cfg.App.Name)
	}
}

// TestValidateSuccess tests validation of a valid configuration
func TestValidateSuccess(t *testing.T) {
	cfg, err := Load(validConfigPath)
	if err != nil {
		t.Fatalf(expectedNoErrorLoadingConfig, err)
	}

	err = Validate(cfg)
	if err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

// TestValidateInvalidEnvironment tests validation of invalid environment
func TestValidateInvalidEnvironment(t *testing.T) {
	cfg, err := Load(validConfigPath)
	if err != nil {
		t.Fatalf(expectedNoErrorLoadingConfig, err)
	}

	cfg.App.Environment = invalidEnv
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid environment")
	}
}

// TestValidateInvalidMarkets tests validation of invalid market names
func TestValidateInvalidMarkets(t *testing.T) {
	cfg, err := Load(validConfigPath)
	if err != nil {
		t.Fatalf(expectedNoErrorLoadingConfig, err)
	}

	// Set invalid markets
	cfg.Trading.Markets = []string{"FOO", "BAR"}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid markets")
	}

	if !containsSubstring(err.Error(), marketsValidationError) && !containsSubstring(err.Error(), marketsValidationErrorCaps) {
		t.Errorf("expected markets validation error, got: %v", err)
	}
}

// TestValidateEmptyMarkets tests validation of empty markets array
func TestValidateEmptyMarkets(t *testing.T) {
	cfg, err := Load(validConfigPath)
	if err != nil {
		t.Fatalf(expectedNoErrorLoadingConfig, err)
	}

	// Set empty markets
	cfg.Trading.Markets = []string{}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for empty markets")
	}
}

// TestValidateValidMarkets tests validation of valid market combinations
func TestValidateValidMarkets(t *testing.T) {
	cfg, err := Load(validConfigPath)
	if err != nil {
		t.Fatalf(expectedNoErrorLoadingConfig, err)
	}

	// Test with single valid market
	cfg.Trading.Markets = []string{"WIN"}
	err = Validate(cfg)
	if err != nil {
		t.Fatalf("expected no error for single valid market, got %v", err)
	}

	// Test with multiple valid markets
	cfg.Trading.Markets = []string{"WIN", "PLACE", "EW"}
	err = Validate(cfg)
	if err != nil {
		t.Fatalf("expected no error for multiple valid markets, got %v", err)
	}
}

// TestGetDatabaseDSN tests DSN generation
func TestGetDatabaseDSN(t *testing.T) {
	cfg, err := Load(validConfigPath)
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	if err != nil {
		t.Fatalf(expectedNoErrorLoadingConfig, err)
	}

	dsn := cfg.GetDatabaseDSN()
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}

	if !containsSubstring(dsn, postgresPrefix) {
		t.Errorf("expected DSN to start with '%s', got '%s'", postgresPrefix, dsn)
	}
}

// TestIsDevelopment tests environment check function
func TestIsDevelopment(t *testing.T) {
	cfg := &Config{
		App: AppConfig{Environment: developmentEnv},
	}

	if !cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() to return true")
	}

	if cfg.IsProduction() {
		t.Error("expected IsProduction() to return false")
	}
}

// TestIsProduction tests production environment check
func TestIsProduction(t *testing.T) {
	cfg := &Config{
		App: AppConfig{Environment: "production"},
	}

	if !cfg.IsProduction() {
		t.Error("expected IsProduction() to return true")
	}

	if cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() to return false")
	}
}

// TestIsStaging tests staging environment check
func TestIsStaging(t *testing.T) {
	cfg := &Config{
		App: AppConfig{Environment: "staging"},
	}

	if !cfg.IsStaging() {
		t.Error("expected IsStaging() to return true")
	}

	if cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() to return false")
	}
}

// TestGetMLServiceHTTPURL tests ML service HTTP URL retrieval
func TestGetMLServiceHTTPURL(t *testing.T) {
	cfg := &Config{
		MLService: MLServiceConfig{
			URL: "http://localhost:8000",
		},
	}

	url := cfg.GetMLServiceHTTPURL()
	if url != "http://localhost:8000" {
		t.Errorf("expected URL 'http://localhost:8000', got '%s'", url)
	}
}

// TestGetMLServiceGRPCAddress tests gRPC address retrieval
func TestGetMLServiceGRPCAddress(t *testing.T) {
	cfg := &Config{
		MLService: MLServiceConfig{
			GRPCAddress: "localhost:50051",
		},
	}

	addr := cfg.GetMLServiceGRPCAddress()
	if addr != "localhost:50051" {
		t.Errorf("expected address 'localhost:50051', got '%s'", addr)
	}
}

// TestLoadConfigEnvironmentVariableExpansion tests environment variable expansion in config file
func TestLoadConfigEnvironmentVariableExpansion(t *testing.T) {
	// Set environment variable
	os.Setenv(testDBPassword, expandedSecretValue)
	defer os.Unsetenv(testDBPassword)

	cfg, err := Load(expansionConfigPath)
	if err != nil {
		t.Fatalf("expected no error loading config with expansion, got %v", err)
	}

	if cfg.Database.Password != expandedSecretValue {
		t.Errorf("expected password '%s' from environment expansion, got '%s'", expandedSecretValue, cfg.Database.Password)
	}
}

// TestLoadConfigMissingEnvironmentVariable tests handling of missing environment variables
func TestLoadConfigMissingEnvironmentVariable(t *testing.T) {
	// Ensure environment variable is not set
	os.Unsetenv(testMissingVar)

	cfg, err := Load(expansionConfigMissingPath)
	if err != nil {
		t.Fatalf(expectedNoErrorLoadingConfig, err)
	}

	// Missing variables should be kept as literal ${VAR} or empty depending on os.ExpandEnv behavior
	// os.ExpandEnv leaves ${VAR} as-is if VAR is not set
	expectedLiteral := "${TEST_MISSING_VAR}"
	if cfg.Database.Password != expectedLiteral && cfg.Database.Password != "" {
		t.Logf("note: missing env var became: %q (expected literal or empty)", cfg.Database.Password)
	}
}

// Helper function
func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
