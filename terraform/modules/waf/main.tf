locals {
  tags = merge({
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }, var.tags)
}

resource "aws_wafv2_ip_set" "allowlist" {
  name               = "${var.project_name}-${var.environment}-allowlist"
  scope              = "REGIONAL"
  ip_address_version = "IPV4"
  addresses          = var.ip_allowlist
  tags               = local.tags
}

resource "aws_wafv2_ip_set" "blocklist" {
  name               = "${var.project_name}-${var.environment}-blocklist"
  scope              = "REGIONAL"
  ip_address_version = "IPV4"
  addresses          = var.ip_blocklist
  tags               = local.tags
}

resource "aws_wafv2_web_acl" "this" {
  name  = "${var.project_name}-${var.environment}-waf"
  scope = "REGIONAL"

  default_action {
    allow {
    }
  }

  rule {
    name     = "rate-limit"
    priority = 1
    action {
      block {
      }
    }
    statement {
      rate_based_statement {
        limit              = var.rate_limit_threshold
        aggregate_key_type = "IP"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "rate-limit"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "ip-reputation"
    priority = 2
    override_action {
      none {
      }
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesAmazonIpReputationList"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "ip-reputation"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "common-rule-set"
    priority = 3
    override_action {
      none {
      }
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesCommonRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "common-rule-set"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "known-bad-inputs"
    priority = 4
    override_action {
      none {
      }
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesKnownBadInputsRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "known-bad-inputs"
      sampled_requests_enabled   = true
    }
  }

  dynamic "rule" {
    for_each = var.enable_geo_blocking ? [1] : []
    content {
      name     = "geo-blocking"
      priority = 5
      action {
        block {
        }
      }
      statement {
        geo_match_statement {
          country_codes = var.allowed_countries
        }
      }
      visibility_config {
        cloudwatch_metrics_enabled = true
        metric_name                = "geo-blocking"
        sampled_requests_enabled   = true
      }
    }
  }

  rule {
    name     = "allowlist"
    priority = 6
    action {
      allow {
      }
    }
    statement {
      ip_set_reference_statement {
        arn = aws_wafv2_ip_set.allowlist.arn
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "allowlist"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "blocklist"
    priority = 7
    action {
      block {
      }
    }
    statement {
      ip_set_reference_statement {
        arn = aws_wafv2_ip_set.blocklist.arn
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "blocklist"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "${var.project_name}-${var.environment}-waf"
    sampled_requests_enabled   = true
  }

  tags = local.tags
}

resource "aws_s3_bucket" "waf_logs" {
  count  = var.enable_logging ? 1 : 0
  bucket = "${var.project_name}-${var.environment}-waf-logs"
  tags   = local.tags
}

resource "aws_s3_bucket_lifecycle_configuration" "waf_logs" {
  count  = var.enable_logging ? 1 : 0
  bucket = aws_s3_bucket.waf_logs[0].id

  rule {
    id     = "expire-logs"
    status = "Enabled"

    expiration {
      days = var.log_retention_days
    }
  }
}

data "aws_iam_policy_document" "firehose_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["firehose.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "firehose" {
  count              = var.enable_logging ? 1 : 0
  name               = "${var.project_name}-${var.environment}-waf-firehose"
  assume_role_policy = data.aws_iam_policy_document.firehose_assume.json
  tags               = local.tags
}

data "aws_iam_policy_document" "firehose_policy" {
  statement {
    actions = ["s3:PutObject", "s3:ListBucket", "s3:GetBucketLocation"]
    resources = [
      aws_s3_bucket.waf_logs[0].arn,
      "${aws_s3_bucket.waf_logs[0].arn}/*"
    ]
  }
}

resource "aws_iam_role_policy" "firehose" {
  count  = var.enable_logging ? 1 : 0
  name   = "${var.project_name}-${var.environment}-waf-firehose"
  role   = aws_iam_role.firehose[0].id
  policy = data.aws_iam_policy_document.firehose_policy.json
}

resource "aws_kinesis_firehose_delivery_stream" "waf_logs" {
  count       = var.enable_logging ? 1 : 0
  name        = "${var.project_name}-${var.environment}-waf-logs"
  destination = "extended_s3"

  extended_s3_configuration {
    role_arn           = aws_iam_role.firehose[0].arn
    bucket_arn         = aws_s3_bucket.waf_logs[0].arn
    buffering_interval = 300
    buffering_size     = 5
  }
}

resource "aws_wafv2_web_acl_logging_configuration" "this" {
  count                   = var.enable_logging ? 1 : 0
  resource_arn            = aws_wafv2_web_acl.this.arn
  log_destination_configs = [aws_kinesis_firehose_delivery_stream.waf_logs[0].arn]
}

resource "aws_sns_topic" "waf_alerts" {
  name = "${var.project_name}-${var.environment}-waf-alerts"
  tags = local.tags
}

data "aws_region" "current" {}

resource "aws_cloudwatch_metric_alarm" "blocked_requests" {
  alarm_name          = "${var.project_name}-${var.environment}-waf-blocked"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "BlockedRequests"
  namespace           = "AWS/WAFV2"
  period              = 300
  statistic           = "Sum"
  threshold           = 100
  alarm_actions       = [aws_sns_topic.waf_alerts.arn]

  dimensions = {
    WebACL = aws_wafv2_web_acl.this.name
    Rule   = "rate-limit"
    Region = data.aws_region.current.name
  }
}

resource "aws_cloudwatch_metric_alarm" "rate_limit" {
  alarm_name          = "${var.project_name}-${var.environment}-waf-rate-limit"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "BlockedRequests"
  namespace           = "AWS/WAFV2"
  period              = 300
  statistic           = "Sum"
  threshold           = 10
  alarm_actions       = [aws_sns_topic.waf_alerts.arn]

  dimensions = {
    WebACL = aws_wafv2_web_acl.this.name
    Rule   = "rate-limit"
    Region = data.aws_region.current.name
  }
}
