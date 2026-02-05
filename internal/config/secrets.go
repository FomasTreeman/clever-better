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

const (
	racingPostSourceName         = "racing_post"
	errLoadAWSConfig             = "failed to load AWS config: %w"
	errGetSecretFromAWSSecrets   = "failed to get secret from AWS Secrets Manager: %w"
	errParseSecretJSON           = "failed to parse secret JSON: %w"
	errParseSecretBinary         = "failed to parse secret binary: %w"
	errNoSecretDataFound         = "no secret data found in AWS Secrets Manager"
)

// SecretsOverlay represents the structure of secrets stored in AWS Secrets Manager
type SecretsOverlay struct {
	DatabasePassword  string `json:"database_password"`
	BetfairAppKey    string `json:"betfair_app_key"`
	BetfairUsername  string `json:"betfair_username"`
	BetfairPassword  string `json:"betfair_password"`
	RacingPostAPIKey string `json:"racing_post_api_key"`
}

// fetchSecretsFromAWS retrieves secrets from AWS Secrets Manager
func fetchSecretsFromAWS(ctx context.Context, region string, secretName string) (*SecretsOverlay, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf(errLoadAWSConfig, err)
	}

	client := secretsmanager.NewFromConfig(awsCfg)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf(errGetSecretFromAWSSecrets, err)
	}

	secrets, err := parseSecretData(result)
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

// parseSecretData parses secret data from AWS response
func parseSecretData(result *secretsmanager.GetSecretValueOutput) (*SecretsOverlay, error) {
	var secrets SecretsOverlay
	if result.SecretString != nil {
		if err := json.Unmarshal([]byte(*result.SecretString), &secrets); err != nil {
			return nil, fmt.Errorf(errParseSecretJSON, err)
		}
	} else if result.SecretBinary != nil {
		if err := json.Unmarshal(result.SecretBinary, &secrets); err != nil {
			return nil, fmt.Errorf(errParseSecretBinary, err)
		}
	} else {
		return nil, fmt.Errorf(errNoSecretDataFound)
	}
	return &secrets, nil
}

// overlaySecretsOnConfig applies secrets to configuration
func overlaySecretsOnConfig(cfg *Config, secrets *SecretsOverlay) {
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

	if secrets.RacingPostAPIKey != "" {
		for i, source := range cfg.DataIngestion.Sources {
			if source.Name == racingPostSourceName {
				cfg.DataIngestion.Sources[i].APIKey = secrets.RacingPostAPIKey
			}
		}
	}
}

// LoadSecretsFromAWS retrieves secrets from AWS Secrets Manager and overlays them onto the configuration
func LoadSecretsFromAWS(cfg *Config, region string, secretName string) error {
	ctx := context.Background()

	secrets, err := fetchSecretsFromAWS(ctx, region, secretName)
	if err != nil {
		return err
	}

	overlaySecretsOnConfig(cfg, secrets)
	return nil
}

// GetSecretsFromAWS retrieves raw secrets from AWS Secrets Manager without applying them
func GetSecretsFromAWS(region string, secretName string) (*SecretsOverlay, error) {
	ctx := context.Background()
	return fetchSecretsFromAWS(ctx, region, secretName)
}

