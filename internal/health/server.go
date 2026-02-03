// Package health provides a lightweight HTTP server for container health checks.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DatabasePinger defines the interface for checking database connectivity.
type DatabasePinger interface {
	Ping(ctx context.Context) error
}

// HealthResponse represents the JSON response for health check endpoints.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp,omitempty"`
	Version   string `json:"version,omitempty"`
	Commit    string `json:"commit,omitempty"`
}

// ReadyResponse represents the JSON response for readiness check endpoints.
type ReadyResponse struct {
	Status   string            `json:"status"`
	Service  string            `json:"service"`
	Checks   map[string]string `json:"checks,omitempty"`
	Duration string            `json:"duration,omitempty"`
}

// Server is a lightweight HTTP server for health check endpoints.
type Server struct {
	serviceName string
	version     string
	commit      string
	port        string
	server      *http.Server
	logger      *logrus.Logger
	db          DatabasePinger
	mu          sync.RWMutex
	ready       bool
}

// Config holds the configuration for the health server.
type Config struct {
	ServiceName string
	Version     string
	Commit      string
	Port        string
	Logger      *logrus.Logger
	DB          DatabasePinger
}

// NewServer creates a new health check server.
func NewServer(cfg Config) *Server {
	port := cfg.Port
	if port == "" {
		port = os.Getenv("HEALTH_PORT")
	}
	if port == "" {
		port = "8080"
	}

	return &Server{
		serviceName: cfg.ServiceName,
		version:     cfg.Version,
		commit:      cfg.Commit,
		port:        port,
		logger:      cfg.Logger,
		db:          cfg.DB,
		ready:       false,
	}
}

// SetReady marks the server as ready to accept traffic.
func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = ready
}

// IsReady returns whether the server is ready.
func (s *Server) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

// Start starts the health check server in the background.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/live", s.handleLive)

	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		if s.logger != nil {
			s.logger.WithFields(logrus.Fields{
				"port":    s.port,
				"service": s.serviceName,
			}).Info("Health check server starting")
		}

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if s.logger != nil {
				s.logger.WithError(err).Error("Health check server error")
			}
		}
	}()

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		s.Shutdown()
	}()

	return nil
}

// Shutdown gracefully shuts down the health check server.
func (s *Server) Shutdown() error {
	if s.server == nil {
		return nil
	}

	if s.logger != nil {
		s.logger.Info("Health check server shutting down")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// handleHealth handles the /health endpoint - basic liveness check.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Service:   s.serviceName,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   s.version,
		Commit:    s.commit,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleLive handles the /live endpoint - kubernetes liveness probe.
func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "ok",
		Service: s.serviceName,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleReady handles the /ready endpoint - checks database connectivity.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	checks := make(map[string]string)
	allHealthy := true

	// Check if manually marked as not ready
	if !s.IsReady() {
		allHealthy = false
		checks["service"] = "not_ready"
	} else {
		checks["service"] = "ok"
	}

	// Check database connectivity if available
	if s.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := s.db.Ping(ctx); err != nil {
			allHealthy = false
			checks["database"] = fmt.Sprintf("error: %v", err)
		} else {
			checks["database"] = "ok"
		}
	}

	response := ReadyResponse{
		Service:  s.serviceName,
		Checks:   checks,
		Duration: time.Since(start).String(),
	}

	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		response.Status = "ok"
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = "not_ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}
