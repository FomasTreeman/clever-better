package betfair

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks Betfair operations
type Metrics struct {
	// API request metrics
	APIRequestsTotal     int64
	APIRequestsSuccess   int64
	APIRequestsFailure   int64
	APIRequestLatency    []time.Duration
	latencyMu            sync.Mutex

	// Market data metrics
	MessagesReceived     int64
	MessageProcessErrors int64
	OddsSnapshotsStored  int64
	BufferFlushes        int64

	// Betting metrics
	BetsPlaced           int64
	BetsMatched          int64
	BetsSettled          int64
	BetsCancelled        int64
	BetPlacementErrors   int64
	AverageBetLatency    time.Duration

	// Session metrics
	AuthenticationFailures int64
	SessionRefreshes       int64
	SessionExpiries        int64
	CurrentSessionValid    bool

	// Stream metrics
	StreamConnections    int64
	StreamDisconnections int64
	StreamReconnections  int64
	LastHeartbeat        time.Time

	mu sync.RWMutex
}

var globalMetrics = &Metrics{}

// RecordAPIRequest records an API request
func RecordAPIRequest(latency time.Duration, success bool) {
	atomic.AddInt64(&globalMetrics.APIRequestsTotal, 1)

	if success {
		atomic.AddInt64(&globalMetrics.APIRequestsSuccess, 1)
	} else {
		atomic.AddInt64(&globalMetrics.APIRequestsFailure, 1)
	}

	globalMetrics.latencyMu.Lock()
	globalMetrics.APIRequestLatency = append(globalMetrics.APIRequestLatency, latency)
	globalMetrics.latencyMu.Unlock()
}

// RecordMessageReceived records a received stream message
func RecordMessageReceived() {
	atomic.AddInt64(&globalMetrics.MessagesReceived, 1)
}

// RecordMessageProcessError records a message processing error
func RecordMessageProcessError() {
	atomic.AddInt64(&globalMetrics.MessageProcessErrors, 1)
}

// RecordOddsSnapshot records stored odds snapshot
func RecordOddsSnapshot() {
	atomic.AddInt64(&globalMetrics.OddsSnapshotsStored, 1)
}

// RecordBufferFlush records a buffer flush operation
func RecordBufferFlush() {
	atomic.AddInt64(&globalMetrics.BufferFlushes, 1)
}

// RecordBetPlaced records a placed bet
func RecordBetPlaced(latency time.Duration, success bool) {
	if success {
		atomic.AddInt64(&globalMetrics.BetsPlaced, 1)
		globalMetrics.mu.Lock()
		globalMetrics.AverageBetLatency = latency
		globalMetrics.mu.Unlock()
	} else {
		atomic.AddInt64(&globalMetrics.BetPlacementErrors, 1)
	}
}

// RecordBetMatched records a matched bet
func RecordBetMatched() {
	atomic.AddInt64(&globalMetrics.BetsMatched, 1)
}

// RecordBetSettled records a settled bet
func RecordBetSettled() {
	atomic.AddInt64(&globalMetrics.BetsSettled, 1)
}

// RecordBetCancelled records a cancelled bet
func RecordBetCancelled() {
	atomic.AddInt64(&globalMetrics.BetsCancelled, 1)
}

// RecordAuthenticationFailure records an authentication failure
func RecordAuthenticationFailure() {
	atomic.AddInt64(&globalMetrics.AuthenticationFailures, 1)
}

// RecordSessionRefresh records a session refresh
func RecordSessionRefresh() {
	atomic.AddInt64(&globalMetrics.SessionRefreshes, 1)
}

// RecordStreamConnection records a stream connection
func RecordStreamConnection() {
	atomic.AddInt64(&globalMetrics.StreamConnections, 1)
}

// RecordStreamDisconnection records a stream disconnection
func RecordStreamDisconnection() {
	atomic.AddInt64(&globalMetrics.StreamDisconnections, 1)
}

// RecordStreamReconnection records a stream reconnection
func RecordStreamReconnection() {
	atomic.AddInt64(&globalMetrics.StreamReconnections, 1)
}

// RecordHeartbeat records a heartbeat
func RecordHeartbeat() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.LastHeartbeat = time.Now()
}

// GetMetrics returns a snapshot of current metrics
func GetMetrics() Metrics {
	globalMetrics.mu.RLock()
	defer globalMetrics.mu.RUnlock()

	return Metrics{
		APIRequestsTotal:       atomic.LoadInt64(&globalMetrics.APIRequestsTotal),
		APIRequestsSuccess:     atomic.LoadInt64(&globalMetrics.APIRequestsSuccess),
		APIRequestsFailure:     atomic.LoadInt64(&globalMetrics.APIRequestsFailure),
		MessagesReceived:        atomic.LoadInt64(&globalMetrics.MessagesReceived),
		MessageProcessErrors:    atomic.LoadInt64(&globalMetrics.MessageProcessErrors),
		OddsSnapshotsStored:     atomic.LoadInt64(&globalMetrics.OddsSnapshotsStored),
		BufferFlushes:           atomic.LoadInt64(&globalMetrics.BufferFlushes),
		BetsPlaced:              atomic.LoadInt64(&globalMetrics.BetsPlaced),
		BetsMatched:             atomic.LoadInt64(&globalMetrics.BetsMatched),
		BetsSettled:             atomic.LoadInt64(&globalMetrics.BetsSettled),
		BetsCancelled:           atomic.LoadInt64(&globalMetrics.BetsCancelled),
		BetPlacementErrors:      atomic.LoadInt64(&globalMetrics.BetPlacementErrors),
		AverageBetLatency:       globalMetrics.AverageBetLatency,
		AuthenticationFailures:  atomic.LoadInt64(&globalMetrics.AuthenticationFailures),
		SessionRefreshes:        atomic.LoadInt64(&globalMetrics.SessionRefreshes),
		CurrentSessionValid:     globalMetrics.CurrentSessionValid,
		StreamConnections:       atomic.LoadInt64(&globalMetrics.StreamConnections),
		StreamDisconnections:    atomic.LoadInt64(&globalMetrics.StreamDisconnections),
		StreamReconnections:     atomic.LoadInt64(&globalMetrics.StreamReconnections),
		LastHeartbeat:           globalMetrics.LastHeartbeat,
	}
}

// ResetMetrics resets all metrics
func ResetMetrics() {
	atomic.StoreInt64(&globalMetrics.APIRequestsTotal, 0)
	atomic.StoreInt64(&globalMetrics.APIRequestsSuccess, 0)
	atomic.StoreInt64(&globalMetrics.APIRequestsFailure, 0)
	atomic.StoreInt64(&globalMetrics.MessagesReceived, 0)
	atomic.StoreInt64(&globalMetrics.MessageProcessErrors, 0)
	atomic.StoreInt64(&globalMetrics.OddsSnapshotsStored, 0)
	atomic.StoreInt64(&globalMetrics.BufferFlushes, 0)
	atomic.StoreInt64(&globalMetrics.BetsPlaced, 0)
	atomic.StoreInt64(&globalMetrics.BetsMatched, 0)
	atomic.StoreInt64(&globalMetrics.BetsSettled, 0)
	atomic.StoreInt64(&globalMetrics.BetsCancelled, 0)
	atomic.StoreInt64(&globalMetrics.BetPlacementErrors, 0)
	atomic.StoreInt64(&globalMetrics.AuthenticationFailures, 0)
	atomic.StoreInt64(&globalMetrics.SessionRefreshes, 0)
	atomic.StoreInt64(&globalMetrics.StreamConnections, 0)
	atomic.StoreInt64(&globalMetrics.StreamDisconnections, 0)
	atomic.StoreInt64(&globalMetrics.StreamReconnections, 0)

	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.APIRequestLatency = nil
	globalMetrics.AverageBetLatency = 0
	globalMetrics.LastHeartbeat = time.Time{}
}
