package service

import (
	"fmt"
	"sync"
	"time"
)

// IngestionMetrics tracks statistics about data ingestion
type IngestionMetrics struct {
	mu                  sync.RWMutex
	StartTime           time.Time
	Duration            time.Duration
	TotalRaces          int
	SuccessfulRaces     int
	TotalRunners        int
	Duplicates          int
	ValidationErrors    int
	Errors              int
}

// NewIngestionMetrics creates a new metrics tracker
func NewIngestionMetrics() *IngestionMetrics {
	return &IngestionMetrics{
		StartTime: time.Now(),
	}
}

// Reset resets all metrics
func (m *IngestionMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartTime = time.Now()
	m.Duration = 0
	m.TotalRaces = 0
	m.SuccessfulRaces = 0
	m.TotalRunners = 0
	m.Duplicates = 0
	m.ValidationErrors = 0
	m.Errors = 0
}

// RecordRace increments successful race count
func (m *IngestionMetrics) RecordRace() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SuccessfulRaces++
}

// RecordRunner increments runner count
func (m *IngestionMetrics) RecordRunner() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalRunners++
}

// RecordDuplicate increments duplicate count
func (m *IngestionMetrics) RecordDuplicate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Duplicates++
}

// RecordError increments error count
func (m *IngestionMetrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Errors++
}

// RecordValidationError increments validation error count
func (m *IngestionMetrics) RecordValidationError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ValidationErrors++
}

// String returns a formatted string representation of metrics
func (m *IngestionMetrics) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	successRate := float64(0)
	if m.TotalRaces > 0 {
		successRate = float64(m.SuccessfulRaces) / float64(m.TotalRaces) * 100
	}

	return fmt.Sprintf(
		"IngestionMetrics{Total=%d, Successful=%d (%.1f%%), Runners=%d, Duplicates=%d, ValidationErrors=%d, Errors=%d, Duration=%v}",
		m.TotalRaces,
		m.SuccessfulRaces,
		successRate,
		m.TotalRunners,
		m.Duplicates,
		m.ValidationErrors,
		m.Errors,
		m.Duration,
	)
}

// ToPrometheus returns metrics in Prometheus format
func (m *IngestionMetrics) ToPrometheus() string {
	m.mu.RLock()
	defer m.mu.Unlock()

	return fmt.Sprintf(
		`# HELP ingestion_total_races Total number of races ingested
# TYPE ingestion_total_races counter
ingestion_total_races{status="total"} %d
ingestion_total_races{status="successful"} %d
ingestion_total_races{status="duplicate"} %d

# HELP ingestion_total_runners Total number of runners ingested
# TYPE ingestion_total_runners counter
ingestion_total_runners %d

# HELP ingestion_errors Total ingestion errors
# TYPE ingestion_errors counter
ingestion_errors{type="validation"} %d
ingestion_errors{type="system"} %d

# HELP ingestion_duration_seconds Last ingestion duration
# TYPE ingestion_duration_seconds gauge
ingestion_duration_seconds %.2f
`,
		m.TotalRaces,
		m.SuccessfulRaces,
		m.Duplicates,
		m.TotalRunners,
		m.ValidationErrors,
		m.Errors,
		m.Duration.Seconds(),
	)
}
