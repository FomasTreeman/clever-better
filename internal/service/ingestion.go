package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/datasource"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

// IngestionService handles the data ingestion workflow
type IngestionService struct {
	sources   []datasource.DataSource
	raceRepo  repository.RaceRepository
	runnerRepo repository.RunnerRepository
	validator *DataValidator
	normalizer *DataNormalizer
	metrics   *IngestionMetrics
	logger    *log.Logger
	batchSize int
}

// NewIngestionService creates a new ingestion service
func NewIngestionService(
	sources []datasource.DataSource,
	raceRepo repository.RaceRepository,
	runnerRepo repository.RunnerRepository,
	validator *DataValidator,
	normalizer *DataNormalizer,
	logger *log.Logger,
	batchSize int,
) *IngestionService {
	if batchSize <= 0 {
		batchSize = 100
	}

	return &IngestionService{
		sources:    sources,
		raceRepo:   raceRepo,
		runnerRepo: runnerRepo,
		validator:  validator,
		normalizer: normalizer,
		metrics:    NewIngestionMetrics(),
		logger:     logger,
		batchSize:  batchSize,
	}
}

// IngestHistoricalData fetches and ingests historical data from a specific source
func (s *IngestionService) IngestHistoricalData(ctx context.Context, sourceName string, startDate, endDate time.Time) (*IngestionMetrics, error) {
	s.metrics.Reset()
	startTime := time.Now()

	s.logger.Printf("Starting historical data ingestion from %s (%s to %s)", sourceName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// Find the specified data source
	var source datasource.DataSource
	for _, src := range s.sources {
		if src.Name() == sourceName {
			source = src
			break
		}
	}

	if source == nil {
		return nil, fmt.Errorf("data source not found: %s", sourceName)
	}

	// Fetch races
	races, err := source.FetchRaces(ctx, startDate, endDate)
	if err != nil {
		s.metrics.Errors++
		s.logger.Printf("Failed to fetch races from %s: %v", sourceName, err)
		return s.metrics, fmt.Errorf("failed to fetch races: %w", err)
	}

	s.logger.Printf("Fetched %d races from %s", len(races), sourceName)
	s.metrics.TotalRaces = len(races)

	// Process races in batches
	for i := 0; i < len(races); i += s.batchSize {
		end := i + s.batchSize
		if end > len(races) {
			end = len(races)
		}

		batch := races[i:end]
		if err := s.processBatch(ctx, batch); err != nil {
			s.logger.Printf("Error processing batch: %v", err)
			s.metrics.Errors++
			// Continue processing other batches
		}
	}

	s.metrics.Duration = time.Since(startTime)
	s.logger.Printf("Historical ingestion complete: %d races, %d runners, %d errors, duration: %v",
		s.metrics.SuccessfulRaces, s.metrics.TotalRunners, s.metrics.Errors, s.metrics.Duration)

	return s.metrics, nil
}

// IngestLiveData fetches and ingests upcoming/live races
func (s *IngestionService) IngestLiveData(ctx context.Context, sourceName string) error {
	// Find the specified data source
	var source datasource.DataSource
	for _, src := range s.sources {
		if src.Name() == sourceName {
			source = src
			break
		}
	}

	if source == nil {
		return fmt.Errorf("data source not found: %s", sourceName)
	}

	// Fetch upcoming races (next 7 days)
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	races, err := source.FetchRaces(ctx, now, tomorrow)
	if err != nil {
		s.logger.Printf("Failed to fetch live races from %s: %v", sourceName, err)
		return fmt.Errorf("failed to fetch live races: %w", err)
	}

	s.logger.Printf("Fetched %d live races from %s", len(races), sourceName)

	// Process races
	for _, race := range races {
		if err := s.processRace(ctx, &race); err != nil {
			s.logger.Printf("Error processing live race: %v", err)
			s.metrics.Errors++
		}
	}

	return nil
}

// processBatch processes a batch of races
func (s *IngestionService) processBatch(ctx context.Context, races []datasource.RaceData) error {
	for _, race := range races {
		if err := s.processRace(ctx, &race); err != nil {
			s.metrics.Errors++
			s.logger.Printf("Error processing race %s: %v", race.SourceID, err)
			continue
		}
	}
	return nil
}

// processRace processes a single race: validate, normalize, persist
func (s *IngestionService) processRace(ctx context.Context, sourceRace *datasource.RaceData) error {
	// Normalize race data
	race, err := s.normalizer.NormalizeRace(sourceRace)
	if err != nil {
		return fmt.Errorf("failed to normalize race: %w", err)
	}

	// Validate race
	validationErrors := s.validator.ValidateRace(race)
	if len(validationErrors) > 0 {
		s.metrics.ValidationErrors++
		return fmt.Errorf("race validation failed: %v", validationErrors)
	}

	// Check for existing race
	existingRaces, err := s.raceRepo.GetByTrackAndDate(ctx, race.Track, race.ScheduledStart)
	if err == nil && len(existingRaces) > 0 {
		// Race already exists, skip
		s.metrics.Duplicates++
		return nil
	}

	// Validate and process runners
	for _, runner := range race.Runners {
		validationErrors := s.validator.ValidateRunner(runner)
		if len(validationErrors) > 0 {
			s.metrics.ValidationErrors++
			s.logger.Printf("Runner validation failed for %s: %v", runner.Name, validationErrors)
			continue
		}

		runnerValidationErrors := s.validator.ValidateRunnerInRace(runner, race)
		if len(runnerValidationErrors) > 0 {
			s.logger.Printf("Runner validation failed in race context: %v", runnerValidationErrors)
			continue
		}
	}

	// Create race in database
	if err := s.raceRepo.Create(ctx, race); err != nil {
		return fmt.Errorf("failed to create race: %w", err)
	}

	// Create runners
	for _, runner := range race.Runners {
		if err := s.runnerRepo.Create(ctx, runner); err != nil {
			s.logger.Printf("Failed to create runner %s: %v", runner.Name, err)
			s.metrics.Errors++
			continue
		}
		s.metrics.TotalRunners++
	}

	s.metrics.SuccessfulRaces++
	return nil
}

// GetMetrics returns current ingestion metrics
func (s *IngestionService) GetMetrics() *IngestionMetrics {
	return s.metrics
}

// ResetMetrics resets ingestion metrics
func (s *IngestionService) ResetMetrics() {
	s.metrics.Reset()
}
