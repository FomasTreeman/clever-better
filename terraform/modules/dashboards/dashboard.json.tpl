{
  "widgets": [
    {
      "type": "metric",
      "properties": {
        "metrics": [
          [ "clever_better", "current_bankroll", { "stat": "Average" } ],
          [ ".", "daily_pnl", { "stat": "Average" } ],
          [ ".", "total_exposure", { "stat": "Maximum" } ]
        ],
        "period": 300,
        "stat": "Average",
        "region": "${region}",
        "title": "Trading Performance",
        "yAxis": {
          "left": {
            "min": 0
          }
        }
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          [ "clever_better", "active_strategies", { "stat": "Average" } ],
          [ ".", "circuit_breaker_trips_total", { "stat": "Sum" } ]
        ],
        "period": 300,
        "stat": "Average",
        "region": "${region}",
        "title": "Strategy Status"
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          [ "clever_better", "bet_placement_latency_seconds", { "stat": "p95" } ],
          [ ".", "strategy_evaluation_duration_seconds", { "stat": "p95" } ]
        ],
        "period": 300,
        "stat": "Average",
        "region": "${region}",
        "title": "Latency Metrics (P95)"
      }
    },
    {
      "type": "log",
      "properties": {
        "query": "fields @timestamp, strategy_id, decision, confidence | filter event = 'strategy_decision' | stats count() by decision",
        "region": "${region}",
        "title": "Recent Strategy Decisions"
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          [ "clever_better", "bets_placed_total", { "stat": "Sum" } ],
          [ ".", "bets_matched_total", { "stat": "Sum" } ],
          [ ".", "bets_settled_total", { "stat": "Sum" } ]
        ],
        "period": 300,
        "stat": "Sum",
        "region": "${region}",
        "title": "Bet Counts"
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          [ "AWS/ApplicationELB", "TargetResponseTime", { "dimensions": { "LoadBalancer": "${alb_arn_suffix}" } } ],
          [ ".", "RequestCount", { "dimensions": { "LoadBalancer": "${alb_arn_suffix}" } } ]
        ],
        "period": 300,
        "stat": "Average",
        "region": "${region}",
        "title": "ALB Metrics"
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          [ "AWS/RDS", "DatabaseConnections", { "dimensions": { "DBInstanceIdentifier": "${rds_instance_id}" } } ],
          [ ".", "FreeableMemory", { "dimensions": { "DBInstanceIdentifier": "${rds_instance_id}" } } ],
          [ ".", "CPUUtilization", { "dimensions": { "DBInstanceIdentifier": "${rds_instance_id}" } } ]
        ],
        "period": 300,
        "stat": "Average",
        "region": "${region}",
        "title": "RDS Metrics"
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          [ "AWS/ECS", "DesiredTaskCount", { "dimensions": { "ClusterName": "${ecs_cluster_name}", "ServiceName": "clever-better-bot" } } ],
          [ ".", "RunningTaskCount", { "dimensions": { "ClusterName": "${ecs_cluster_name}", "ServiceName": "clever-better-bot" } } ]
        ],
        "period": 300,
        "stat": "Average",
        "region": "${region}",
        "title": "ECS Task Status"
      }
    },
    {
      "type": "log",
      "properties": {
        "query": "fields @timestamp, ml_confidence, recommendation | filter event = 'strategy_evaluation' | stats avg(ml_confidence) as avg_confidence by recommendation",
        "region": "${region}",
        "title": "ML Confidence by Recommendation"
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          [ "clever_better", "strategy_composite_score" ]
        ],
        "period": 300,
        "stat": "Average",
        "region": "${region}",
        "title": "Strategy Composite Scores"
      }
    }
  ]
}
