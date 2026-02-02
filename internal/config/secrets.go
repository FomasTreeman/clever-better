// Package config provides configuration management for the Clever Better application.
package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SecretsOverlay represents the structure of secrets stored in AWS Secrets Manager
type SecretsOverlay struct {
	DatabasePassword  string `json:"database_password"`
	BetfairAppKey    string `json:"betfair_app_key"`
	BetfairUsername  string `json:"betfair_username"`
	BetfairPassword  string `json:"betfair_password"`
	RacingPostAPIKey string `json:"racing_post_api_key"`
}

// LoadSecretsFromAWS retrieves secrets from AWS Secrets Manager and overlays them onto the configuration
func LoadSecretsFromAWS(cfg *Config, region string, secretName string) error {
	ctx := context.Background()

	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(awsCfg)

	// Get secret value
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := client.GetSecretValue(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to get secret from AWS Secrets Manager: %w", err)
	}

	// Parse the secret JSON
	var secrets SecretsOverlay
	if result.SecretString != nil {
		if err := json.Unmarshal([]byte(*result.SecretString), &secrets); err != nil {
			return fmt.Errorf("failed to parse secret JSON: %w", err)
		}
	} else if result.SecretBinary != nil {
		if err := json.Unmarshal(result.SecretBinary, &secrets); err != nil {
			return fmt.Errorf("failed to parse secret binary: %w", err)
		}
	} else {
		return fmt.Errorf("no secret data found in AWS Secrets Manager")
	}

	// Overlay secrets onto configuration
	if secrets.DatabasePassword != "" {
		cfg.Database.Password = secrets.DatabasePassword
	}
	if secrets.BetfairAppKey != "" {
		cfg.Betfair.AppKey = secrets.BetfairAppKey
	}
	if secrets.BetfairUsername != "" {
		cfg.Betfair.Username = secrets.BetfairUsername
	}
	if secrets.BetfairPassword != "" {
		cfg.Betfair.Password = secrets.BetfairPassword
	}

	// Update data ingestion sources with API key if present
	if secrets.RacingPostAPIKey != "" {
		for i, source := range cfg.DataIngestion.Sources {
			if source.Name == "racing_post" {
				cfg.DataIngestion.Sources[i].APIKey = secrets.RacingPostAPIKey
			}
		}
	}

	return nil
}

// GetSecretsFromAWS retrieves raw secrets from AWS Secrets Manager without applying them
func GetSecretsFromAWS(region string, secretName string) (*SecretsOverlay, error) {
	ctx := context.Background()

	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(awsCfg)

	// Get secret value
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret from AWS Secrets Manager: %w", err)
	}

	// Parse the secret JSON
	var secrets SecretsOverlay
	if result.SecretString != nil {
		if err := json.Unmarshal([]byte(*result.SecretString), &secrets); err != nil {
			return nil, fmt.Errorf("failed to parse secret JSON: %w", err)
		}
	} else if result.SecretBinary != nil {
		if err := json.Unmarshal(result.SecretBinary, &secrets); err != nil {
			return nil, fmt.Errorf("failed to parse secret binary: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no secret data found in AWS Secrets Manager")
	}

	return &secrets, nil
}
