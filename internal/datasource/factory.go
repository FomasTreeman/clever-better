package datasource

import (
	"fmt"
	"log"

	"github.com/yourusername/clever-better/internal/config"
)

// SourceType represents the type of data source
type SourceType string

const (
	// Betfair data source type
	BetfairSourceType SourceType = "betfair"
	// Racing Post data source type
	RacingPostSourceType SourceType = "racing_post"
	// CSV file data source type
	CSVSourceType SourceType = "csv"
)

// Factory creates DataSource implementations based on configuration
type Factory struct {
	logger *log.Logger
	config *config.Config
}

// NewFactory creates a new data source factory
func NewFactory(cfg *config.Config, logger *log.Logger) *Factory {
	return &Factory{
		logger: logger,
		config: cfg,
	}
}

// NewDataSource creates a new DataSource based on the provided configuration
func (f *Factory) NewDataSource(cfg config.DataSourceConfig, httpClient *RateLimitedHTTPClient) (DataSource, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("HTTP client is required")
	}

	switch cfg.Name {
	case "betfair_historical":
		return NewBetfairHistoricalClient(httpClient, cfg.APIKey, cfg.Enabled, f.logger), nil

	case "racing_post":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("Racing Post API key is required")
		}
		return NewRacingPostClient(httpClient, cfg.APIKey, cfg.Enabled, f.logger), nil

	default:
		return nil, fmt.Errorf("unknown data source: %s", cfg.Name)
	}
}

// Create creates a new data source based on the type
func (f *Factory) Create(sourceType SourceType) (DataSource, error) {
	switch sourceType {
	case BetfairSourceType:
		return f.createBetfairSource()
	case RacingPostSourceType:
		return f.createRacingPostSource()
	case CSVSourceType:
		return f.createCSVSource()
	default:
		return nil, fmt.Errorf("unknown data source type: %s", sourceType)
	}
}

// createBetfairSource creates a Betfair data source
func (f *Factory) createBetfairSource() (DataSource, error) {
	// For now, return a placeholder that uses legacy creation
	// This will be fully implemented when datasources config is available
	return nil, fmt.Errorf("betfair source creation requires updated config")
}

// createRacingPostSource creates a Racing Post data source
func (f *Factory) createRacingPostSource() (DataSource, error) {
	// For now, return a placeholder that uses legacy creation
	return nil, fmt.Errorf("racing_post source creation requires updated config")
}

// createCSVSource creates a CSV file data source
func (f *Factory) createCSVSource() (DataSource, error) {
	// For now, return a placeholder that uses legacy creation
	return nil, fmt.Errorf("csv source creation requires updated config")
}

// ListAvailableSources returns a list of available source types
func (f *Factory) ListAvailableSources() []SourceType {
	available := make([]SourceType, 0)

	if f.config != nil && f.config.DataSources != nil {
		for sourceType := range f.config.DataSources {
			switch sourceType {
			case "betfair":
				available = append(available, BetfairSourceType)
			case "racing_post":
				available = append(available, RacingPostSourceType)
			case "csv":
				available = append(available, CSVSourceType)
			}
		}
	}

	return available
}

// NewDataSources creates all enabled data sources from configuration
func (f *Factory) NewDataSources(dataCfg config.DataIngestionConfig, httpClient *RateLimitedHTTPClient) ([]DataSource, error) {
	var sources []DataSource

	for _, srcCfg := range dataCfg.Sources {
		if !srcCfg.Enabled {
			if f.logger != nil {
				f.logger.Printf("Skipping disabled data source: %s", srcCfg.Name)
			}
			continue
		}

		source, err := f.NewDataSource(srcCfg, httpClient)
		if err != nil {
			return nil, fmt.Errorf("failed to create data source %s: %w", srcCfg.Name, err)
		}

		sources = append(sources, source)
		if f.logger != nil {
			f.logger.Printf("Created data source: %s", srcCfg.Name)
		}
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no enabled data sources configured")
	}

	return sources, nil
}
