package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/yourusername/clever-better/internal/service"
)

// Scheduler manages scheduled data ingestion jobs
type Scheduler struct {
	cron           *cron.Cron
	ingestionSvc   *service.IngestionService
	logger         *log.Logger
	mu             sync.RWMutex
	isRunning      bool
	jobIDs         []cron.EntryID
	gracefulTimeout time.Duration
}

// NewScheduler creates a new scheduler
func NewScheduler(ingestionSvc *service.IngestionService, logger *log.Logger) *Scheduler {
	return &Scheduler{
		cron:            cron.New(cron.WithLocation(time.UTC)),
		ingestionSvc:    ingestionSvc,
		logger:          logger,
		jobIDs:          make([]cron.EntryID, 0),
		gracefulTimeout: 30 * time.Second,
	}
}

// ScheduleHistoricalSync schedules historical data synchronization
func (s *Scheduler) ScheduleHistoricalSync(cronExpression string, sourceName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("cannot schedule job while scheduler is running")
	}

	jobFunc := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
		defer cancel()

		// Default: sync last 7 days
		endDate := time.Now()
		startDate := endDate.Add(-7 * 24 * time.Hour)

		s.logger.Printf("Starting scheduled historical sync from %s for %s to %s",
			sourceName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

		metrics, err := s.ingestionSvc.IngestHistoricalData(ctx, sourceName, startDate, endDate)
		if err != nil {
			s.logger.Printf("Error during scheduled historical sync: %v", err)
		} else {
			s.logger.Printf("Scheduled historical sync completed: %s", metrics.String())
		}
	}

	entryID, err := s.cron.AddFunc(cronExpression, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add job: %w", err)
	}

	s.jobIDs = append(s.jobIDs, entryID)
	s.logger.Printf("Scheduled historical sync job with cron expression: %s", cronExpression)

	return nil
}

// ScheduleLivePolling schedules live/upcoming race polling
func (s *Scheduler) ScheduleLivePolling(intervalSeconds int, sourceName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("cannot schedule job while scheduler is running")
	}

	if intervalSeconds < 5 {
		intervalSeconds = 5
	}

	jobFunc := func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(intervalSeconds-1)*time.Second)
		defer cancel()

		if err := s.ingestionSvc.IngestLiveData(ctx, sourceName); err != nil {
			s.logger.Printf("Error during live polling from %s: %v", sourceName, err)
		}
	}

	entryID, err := s.cron.AddFunc(fmt.Sprintf("@every %ds", intervalSeconds), jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add job: %w", err)
	}

	s.jobIDs = append(s.jobIDs, entryID)
	s.logger.Printf("Scheduled live polling job with interval: %d seconds", intervalSeconds)

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	if len(s.jobIDs) == 0 {
		return fmt.Errorf("no jobs scheduled")
	}

	s.cron.Start()
	s.isRunning = true
	s.logger.Printf("Scheduler started with %d jobs", len(s.jobIDs))

	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.gracefulTimeout)
	defer cancel()

	<-s.cron.Stop().Done()
	s.isRunning = false
	s.logger.Printf("Scheduler stopped")

	return nil
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// GetNextRun returns the time of the next scheduled job run
func (s *Scheduler) GetNextRun() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isRunning || len(s.jobIDs) == 0 {
		return time.Time{}
	}

	nextRun := time.Time{}
	for _, jobID := range s.jobIDs {
		entry := s.cron.Entry(jobID)
		if entry.Valid() {
			nextTime := entry.Next
			if nextRun.IsZero() || nextTime.Before(nextRun) {
				nextRun = nextTime
			}
		}
	}

	return nextRun
}

// Entries returns information about scheduled entries
func (s *Scheduler) Entries() []cron.Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]cron.Entry, 0, len(s.jobIDs))
	for _, jobID := range s.jobIDs {
		entry := s.cron.Entry(jobID)
		if entry.Valid() {
			entries = append(entries, entry)
		}
	}

	return entries
}

// RemoveJob removes a scheduled job
func (s *Scheduler) RemoveJob(jobID cron.EntryID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("cannot remove job while scheduler is running")
	}

	s.cron.Remove(jobID)
	s.logger.Printf("Removed job: %d", jobID)

	return nil
}
