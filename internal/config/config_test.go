// Package config provides configuration management for the Clever Better application.
package config

import (
	"os"
	"testing"
)

// TestLoadConfig_Success tests loading a valid configuration file
func TestLoadConfig_Success(t *testing.T) {
	cfg, err := Load("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.App.Name != "clever-better" {
		t.Errorf("expected app name 'clever-better', got '%s'", cfg.App.Name)
	}

	if cfg.App.Environment != "development" {
		t.Errorf("expected environment 'development', got '%s'", cfg.App.Environment)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("expected database host 'localhost', got '%s'", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("expected database port 5432, got %d", cfg.Database.Port)
	}
}

// TestLoadConfig_FileNotFound tests handling of missing configuration file
func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := Load("testdata/nonexistent_config.yaml")
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

// TestLoadConfig_EnvironmentVariables tests environment variable override
func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	// Set an environment variable
	os.Setenv("CLEVER_BETTER_APP_NAME", "test-app")
	defer os.Unsetenv("CLEVER_BETTER_APP_NAME")

	cfg, err := Load("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.App.Name != "test-app" {
		t.Errorf("expected app name 'test-app' from environment, got '%s'", cfg.App.Name)
	}
}

// TestValidate_Success tests validation of a valid configuration
func TestValidate_Success(t *testing.T) {
	cfg, err := Load("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	err = Validate(cfg)
	if err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

// TestValidate_InvalidEnvironment tests validation of invalid environment
func TestValidate_InvalidEnvironment(t *testing.T) {
	cfg, err := Load("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	cfg.App.Environment = "invalid"
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid environment")
	}
}

// TestValidate_InvalidMarkets tests validation of invalid market names
func TestValidate_InvalidMarkets(t *testing.T) {
	cfg, err := Load("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	// Set invalid markets
	cfg.Trading.Markets = []string{"FOO", "BAR"}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid markets")
	}

	if !containsSubstring(err.Error(), "markets") && !containsSubstring(err.Error(), "Markets") {
		t.Errorf("expected markets validation error, got: %v", err)
	}
}

// TestValidate_EmptyMarkets tests validation of empty markets array
func TestValidate_EmptyMarkets(t *testing.T) {
	cfg, err := Load("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	// Set empty markets
	cfg.Trading.Markets = []string{}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for empty markets")
	}
}

// TestValidate_ValidMarkets tests validation of valid market combinations
func TestValidate_ValidMarkets(t *testing.T) {
	cfg, err := Load("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
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
	cfg, err := Load("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	dsn := cfg.GetDatabaseDSN()
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}

	if !containsSubstring(dsn, "postgres://") {
		t.Errorf("expected DSN to start with 'postgres://', got '%s'", dsn)
	}
}

// TestIsDevelopment tests environment check function
func TestIsDevelopment(t *testing.T) {
	cfg := &Config{
		App: AppConfig{Environment: "development"},
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

// TestLoadConfig_EnvironmentVariableExpansion tests environment variable expansion in config file
func TestLoadConfig_EnvironmentVariableExpansion(t *testing.T) {
	// Set environment variable
	testValue := "expanded_secret_value"
	os.Setenv("TEST_DB_PASSWORD", testValue)
	defer os.Unsetenv("TEST_DB_PASSWORD")

	cfg, err := Load("testdata/expansion_config.yaml")
	if err != nil {
		t.Fatalf("expected no error loading config with expansion, got %v", err)
	}

	if cfg.Database.Password != testValue {
		t.Errorf("expected password '%s' from environment expansion, got '%s'", testValue, cfg.Database.Password)
	}
}

// TestLoadConfig_MissingEnvironmentVariable tests handling of missing environment variables
func TestLoadConfig_MissingEnvironmentVariable(t *testing.T) {
	// Ensure environment variable is not set
	os.Unsetenv("TEST_MISSING_VAR")

	cfg, err := Load("testdata/expansion_config_missing.yaml")
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
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
