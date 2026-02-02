// Package main provides the entry point for the data ingestion service.
package main

import (
	"log"
	"os"

	"github.com/yourusername/clever-better/internal/config"
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

	// TODO: Set up logging
	// TODO: Initialize database connection
	// TODO: Initialize data source clients
	// TODO: Start data collection workers
	// TODO: Handle graceful shutdown

	log.Println("Clever Better Data Ingestion Service")
	log.Printf("Running in %s mode with log level: %s", cfg.App.Environment, cfg.App.LogLevel)
	log.Println("Configuration loaded and validated successfully")
}
