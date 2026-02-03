// Package tracing provides AWS X-Ray distributed tracing integration.
package tracing

import (
	"context"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/aws/aws-xray-sdk-go/xraylog"
	"github.com/sirupsen/logrus"
)

// Config contains X-Ray configuration.
type Config struct {
	ServiceName    string
	Enabled        bool
	SamplingRate   float64
	DaemonAddr     string
}

// Logger adapter for X-Ray SDK.
type xrayLoggerAdapter struct {
	logger *logrus.Logger
}

func (l *xrayLoggerAdapter) Log(level xraylog.LogLevel, msg string) {
	switch level {
	case xraylog.LogLevelDebug:
		l.logger.Debug(msg)
	case xraylog.LogLevelInfo:
		l.logger.Info(msg)
	case xraylog.LogLevelWarn:
		l.logger.Warn(msg)
	case xraylog.LogLevelError:
		l.logger.Error(msg)
	}
}

// Initialize initializes AWS X-Ray with the given configuration.
func Initialize(cfg Config, logger *logrus.Logger) error {
	if !cfg.Enabled {
		return nil
	}

	// Set X-Ray logger
	xraylog.SetLogger(&xrayLoggerAdapter{logger: logger})

	// Configure X-Ray
	xray.Configure(xray.Config{
		DaemonAddr:   cfg.DaemonAddr,
		SamplingRate: cfg.SamplingRate,
	})

	logger.WithFields(logrus.Fields{
		"daemon_addr":    cfg.DaemonAddr,
		"sampling_rate":  cfg.SamplingRate,
		"service_name":   cfg.ServiceName,
	}).Info("AWS X-Ray initialized")

	return nil
}

// StartSegment starts a new X-Ray segment.
func StartSegment(ctx context.Context, segmentName string) (context.Context, *xray.Segment) {
	return xray.BeginSegment(ctx, segmentName)
}

// StartSubsegment starts a new X-Ray subsegment.
func StartSubsegment(ctx context.Context, subsegmentName string) (context.Context, *xray.Segment) {
	return xray.BeginSubsegment(ctx, subsegmentName)
}

// AddAnnotation adds an annotation to the current segment.
func AddAnnotation(ctx context.Context, key string, value interface{}) {
	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddAnnotation(key, value)
	}
}

// AddMetadata adds metadata to the current segment.
func AddMetadata(ctx context.Context, key string, value interface{}) {
	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddMetadata(key, value)
	}
}

// AddError adds an error to the current segment.
func AddError(ctx context.Context, err error) {
	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddError(err)
	}
}
