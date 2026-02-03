# Monitoring and Observability Guide

## Overview

The Clever Better system implements comprehensive observability across four pillars:

1. **Prometheus Metrics** - Real-time metric collection from all components
2. **Structured Logging** - Intensive domain-specific logging to CloudWatch
3. **AWS X-Ray Tracing** - Distributed tracing for request flows
4. **CloudWatch Dashboards & Alerts** - Real-time visualization and automated alerting

## Prometheus Metrics

### Architecture

All Prometheus metrics are collected through a centralized registry (`internal/metrics/metrics.go`) with namespace `clever_better`. Metrics are exposed via HTTP endpoint at `/metrics` on port 9090.

### Base Metrics

#### Counter Metrics

Track cumulative counts of events:

| Metric | Labels | Description |
|--------|--------|-------------|
| `clever_better_bets_placed_total` | strategy_id, strategy_name | Total bets placed by strategy |
| `clever_better_bets_matched_total` | strategy_id, strategy_name | Total bets matched on exchange |
| `clever_better_bets_settled_total` | strategy_id, strategy_name | Total bets settled |
| `clever_better_strategy_evaluations_total` | strategy_id, strategy_name | Strategy evaluation cycles |
| `clever_better_strategy_signals_total` | strategy_id, signal_type | Trading signals generated |
| `clever_better_circuit_breaker_trips_total` | reason | Circuit breaker activation events |

#### Gauge Metrics

Track point-in-time measurements:

| Metric | Labels | Description |
|--------|--------|-------------|
| `clever_better_active_strategies` | - | Currently active strategies |
| `clever_better_current_bankroll` | - | Current account bankroll |
| `clever_better_total_exposure` | - | Total market exposure |
| `clever_better_daily_pnl` | - | Daily profit/loss |
| `clever_better_strategy_composite_score` | strategy_id, strategy_name | ML composite score |
| `clever_better_strategy_active_bets` | strategy_id | Active bets per strategy |

#### Histogram Metrics

Track distributions of measurements:

| Metric | Labels | Description |
|--------|--------|-------------|
| `clever_better_bet_placement_latency_seconds` | strategy_id | Time to place bet |
| `clever_better_strategy_evaluation_duration_seconds` | strategy_id | Evaluation cycle duration |
| `clever_better_backtest_duration_seconds` | method | Backtest execution time |

### Strategy-Specific Metrics

Located in `internal/metrics/strategy_metrics.go`:

- `clever_better_strategy_decisions_total[strategy_id, strategy_name, decision_type]` - Strategy decisions by type
- `clever_better_ml_strategy_recommendations_total[strategy_id, strategy_name]` - ML recommendations
- `clever_better_strategy_confidence_score[strategy_id]` - Prediction confidence distribution
- `clever_better_strategy_active_bets[strategy_id]` - Active bets gauge per strategy

### Backtest Metrics

Located in `internal/metrics/backtest_metrics.go`:

- `clever_better_backtest_runs_total[method, status]` - Backtest execution counts
- `clever_better_backtest_composite_score[strategy_id, method]` - Score distribution
- `clever_better_backtest_aggregated_score[strategy_id]` - Aggregated scores

### Integration

#### Recording Events

```go
import "github.com/yourusername/clever-better/internal/metrics"

// Initialize registry on startup
metrics.InitRegistry()

// Record events as they occur
metrics.RecordBetPlaced(strategyID, strategyName, timestamp)
metrics.RecordStrategyEvaluation(strategyID, durationSeconds)
metrics.UpdateBankroll(currentBankroll)
metrics.UpdateExposure(totalExposure)
metrics.RecordCircuitBreakerTrip("max_daily_loss")
```

#### HTTP Endpoint

Prometheus scrapes the `/metrics` endpoint on port 9090:

```yaml
scrape_configs:
  - job_name: 'clever-better-bot'
    static_configs:
      - targets: ['localhost:9090']
```

## Structured Logging

### Three Specialized Loggers

#### Strategy Logger (`internal/logger/strategy_logger.go`)

Focuses on strategy evaluation and trading decisions:

```go
logger := logger.NewStrategyLogger(appLog)

// Log strategy evaluation cycle
logger.LogStrategyEvaluation(
    strategyID, strategyName, runners, signals, duration,
)

// Log trading decision
logger.LogStrategyDecision(
    strategyID, strategyName, decision, confidence,
    edge, kelly, stake, selection, odds,
)

// Log ML filtering
logger.LogMLFiltering(strategyID, beforeCount, afterCount, reason)

// Log strategy lifecycle
logger.LogStrategyActivation(strategyID, strategyName, reason)
logger.LogStrategyDeactivation(strategyID, strategyName, reason)

// Log performance metrics
logger.LogStrategyPnLUpdate(strategyID, pnl, bankroll, wins, losses)
logger.LogStrategyDrawdown(strategyID, drawdown, threshold)
```

#### ML Logger (`internal/logger/ml_logger.go`)

Tracks ML predictions, model training, and backtesting:

```go
logger := logger.NewMLLogger(appLog)

// Log prediction request
logger.LogMLPredictionRequest(modelType, featuresCount, cacheHit, latencyMs)

// Log strategy generation
logger.LogStrategyGeneration(constraints, candidates, score, features)

// Log backtest feedback
logger.LogBacktestFeedback(backtest method, metricsCount, featuresExtracted)

// Log model training
logger.LogModelTraining(modelName, duration, metrics, hyperparameters)

// Log ranking updates
logger.LogStrategyRankingUpdate(totalStrategies, topStrategy, criteria)

// Log errors
logger.LogMLPredictionError(modelType, reason)
```

#### Audit Logger (`internal/logger/audit_logger.go`)

Records all trading actions and system events:

```go
logger := logger.NewAuditLogger(appLog)

// Log bet placement
logger.LogBetPlacement(
    betID, strategyID, market, selection, betType,
    stake, odds, timestamp, paperTrading,
)

// Log bet state changes
logger.LogBetStateChange(betID, oldState, newState, matched, unmatched)

// Log parameter changes
logger.LogStrategyParameterChange(strategyID, paramName, oldValue, newValue, changedBy)

// Log circuit breaker events
logger.LogCircuitBreakerEvent(eventType, reason, metrics, action)

// Log emergency events
logger.LogEmergencyShutdown(reason, systemState)
```

### Log Output Format

All logs are structured JSON output to CloudWatch. Example:

```json
{
  "timestamp": "2024-02-03T12:00:00Z",
  "component": "strategy",
  "strategy_id": "strategy_001",
  "event": "strategy_decision",
  "decision": "PLACE_BET",
  "confidence": 0.87,
  "edge": 0.045,
  "kelly": 0.05,
  "stake": 100,
  "selection": "Market:Runner",
  "odds": 3.5
}
```

### CloudWatch Log Groups

- `/clever-better/strategy-decisions` - Strategy-specific logs
- `/clever-better/ml-operations` - ML predictions and training
- `/clever-better/audit-trail` - Bet placement and system events
- `/clever-better/bot-application` - General application logs
- `/clever-better/backtest-runs` - Backtesting logs

## AWS X-Ray Tracing

### Initialization

```go
import "github.com/yourusername/clever-better/internal/tracing"

// Configure X-Ray on startup
tracing.Initialize(tracing.Config{
    ServiceName: "clever-better-bot",
    Enabled: true,
    SamplingRate: 0.1,
    DaemonAddr: "localhost:2000",
}, logger)
```

### Creating Segments

```go
// Create main segment for trading loop
ctx, segment := tracing.StartSegment(context.Background(), "trading-loop")
defer segment.Close(nil)

// Create subsegments for components
ctx2, subseg := tracing.StartSubsegment(ctx, "strategy-evaluation")
defer subseg.Close(nil)

// Add metadata
tracing.AddAnnotation(ctx, "race_id", raceID)
tracing.AddAnnotation(ctx2, "strategy_id", strategyID)
tracing.AddMetadata(ctx2, "strategy_details", strategyConfig)

// Record errors
if err != nil {
    tracing.AddError(ctx2, err)
}
```

### Sampling Rules

- **Errors**: 100% sampling (all errors captured)
- **Success**: 10% sampling in production (1 in 10 successful requests)
- **Development**: 100% sampling for all traces

### X-Ray Console

Access traces in the [AWS X-Ray console](https://console.aws.amazon.com/xray/):

1. View service map showing component interactions
2. Examine individual traces for slow requests
3. Analyze error traces for debugging
4. Check latency histograms by operation

## CloudWatch Dashboards

### Dashboard Structure

**Trading Performance Section**
- Bankroll trend (last 24h)
- Daily P&L (bar chart)
- Active strategies (gauge)
- Circuit breaker status (text)

**Strategy Performance Section**
- Top 5 strategies by ROI
- Win rate by strategy (table)
- Bet count distribution
- P&L by strategy (stacked bar)

**ML Operations Section**
- Prediction cache hit ratio
- Average prediction latency
- Model accuracy trend
- Training job status

**Backtesting Section**
- Composite scores by method
- Backtest duration distribution
- Strategy ranking (table)
- Confidence scores (histogram)

**Infrastructure Section**
- ECS task count and CPU
- RDS storage and connections
- ALB request count and latency
- CloudWatch log ingestion rate

### Dashboard URL

After Terraform deployment, access the dashboard at:

```
https://console.aws.amazon.com/cloudwatch/home?region=<region>#dashboards:name=clever-better-<environment>-trading
```

### Creating Custom Dashboards

```hcl
# In terraform/environments/<env>/main.tf
module "dashboards" {
  source = "../../modules/dashboards"
  
  environment = var.environment
  project_name = var.project_name
  ecs_cluster_name = module.ecs.cluster_name
  rds_instance_id = module.rds.instance_id
  alb_arn_suffix = module.alb.arn_suffix
  log_group_names = [
    "/clever-better/strategy-decisions",
    "/clever-better/ml-operations",
    "/clever-better/audit-trail"
  ]
}
```

## CloudWatch Alerts

### Alert Configuration

Alarms are configured in `terraform/modules/alerts/`:

| Alarm | Threshold | Severity | Action |
|-------|-----------|----------|--------|
| Daily Loss Exceeded | P&L < -500 | CRITICAL | Pause trading |
| Exposure Limit Exceeded | Total exposure > 50000 | WARNING | Review positions |
| Bankroll Critical Low | Bankroll < 1000 | CRITICAL | Emergency review |
| No Active Strategies | Active strategies = 0 | WARNING | Check system |
| Circuit Breaker Trips | Any trip | INFO | Log and monitor |

### Setting Alert Thresholds

```hcl
module "alerts" {
  source = "../../modules/alerts"
  
  environment = var.environment
  project_name = var.project_name
  
  daily_loss_threshold = -500
  exposure_limit = 50000
  critical_bankroll_threshold = 1000
  
  critical_email = var.critical_alert_email
  warning_email = var.warning_alert_email
  info_email = var.info_alert_email
}
```

### Alert Subscriptions

Alerts are published to SNS topics by severity:

- **Critical Alerts** (`clever-better-<env>-critical-alarms`)
  - Email notifications immediate
  - Slack integration for critical events
  
- **Warning Alerts** (`clever-better-<env>-warning-alarms`)
  - Email notifications (digest mode)
  - Dashboard updates
  
- **Info Alerts** (`clever-better-<env>-info-alarms`)
  - Log file storage
  - Weekly summary reports

## Querying Metrics

### Prometheus Query Examples

```promql
# Average bankroll
avg(clever_better_current_bankroll)

# Win rate over time
rate(clever_better_bets_settled_total[5m])

# P95 bet placement latency
histogram_quantile(0.95, clever_better_bet_placement_latency_seconds)

# Circuit breaker trips today
increase(clever_better_circuit_breaker_trips_total[1d])
```

### CloudWatch Insights Log Queries

```
# Recent strategy decisions
fields @timestamp, strategy_id, decision, confidence
| filter event = "strategy_decision"
| stats avg(confidence) as avg_confidence by strategy_id

# Failed predictions
fields @timestamp, model_type, reason
| filter event = "ml_prediction_error"

# Bet placements today
fields @timestamp, bet_type, stake, odds
| filter event = "bet_placement"
| stats sum(stake) as total_stake, count() as bet_count
```

## Troubleshooting

### No Metrics Appearing

1. Check Prometheus scrape configuration
2. Verify `/metrics` endpoint is responding: `curl localhost:9090/metrics`
3. Check Go application logs for initialization errors
4. Verify IAM permissions for CloudWatch Insights

### Logs Not Appearing in CloudWatch

1. Verify IAM role has `logs:CreateLogGroup` and `logs:PutLogEvents`
2. Check log group names match configuration
3. Verify JSON format validation in loggers
4. Check network connectivity to CloudWatch

### X-Ray Traces Missing

1. Verify X-Ray daemon is running: `curl localhost:2000`
2. Check sampling rules are not excluding traces
3. Verify IAM role has `xray:PutTraceSegments` and `xray:PutTelemetryRecords`
4. Check application is calling `tracing.Initialize()`

### High Alert Noise

1. Review threshold values in `terraform/modules/alerts/variables.tf`
2. Adjust SNS subscription filters
3. Implement exponential backoff in alert generation
4. Create composite alarms to reduce duplicates

## Best Practices

### Metrics

1. **Use appropriate metric types** - Counters for cumulative counts, gauges for point-in-time
2. **Label consistently** - Use strategy_id and strategy_name for all strategy metrics
3. **Monitor cardinality** - Avoid unbounded label values
4. **Set retention** - Configure Prometheus retention to match needs

### Logging

1. **Use correct logger** - StrategyLogger for decisions, MLLogger for predictions, AuditLogger for trades
2. **Include correlation IDs** - Trace requests through components
3. **Avoid PII** - Don't log sensitive customer data
4. **Set appropriate log levels** - DEBUG for detailed, INFO for events, WARN for issues

### Tracing

1. **Create subsegments** - Track nested operations
2. **Add annotations** - Record business context
3. **Sample appropriately** - Balance observability with cost
4. **Monitor X-Ray costs** - Adjust sampling rates as needed

### Dashboards

1. **Keep dashboards focused** - One dashboard per role/audience
2. **Use refresh rates** - 1 minute for operational, 5 minutes for analysis
3. **Add runbooks** - Link to documentation from dashboard
4. **Version control** - Store dashboards in Terraform, not CloudWatch UI

## Related Documentation

- [Architecture Overview](ARCHITECTURE.md)
- [Deployment Guide](DEPLOYMENT.md)
- [AWS X-Ray Documentation](https://docs.aws.amazon.com/xray/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [CloudWatch Documentation](https://docs.aws.amazon.com/cloudwatch/)
