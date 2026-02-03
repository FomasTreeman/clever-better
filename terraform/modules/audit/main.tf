# Audit Module - CloudTrail and GuardDuty
# Provides comprehensive audit logging and threat detection

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Data sources
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}
data "aws_partition" "current" {}

# Local variables
locals {
  name_prefix = "${var.project_name}-${var.environment}"
  account_id  = data.aws_caller_identity.current.account_id
  
  common_tags = merge(
    var.tags,
    {
      Name        = "${local.name_prefix}-audit"
      Environment = var.environment
      ManagedBy   = "terraform"
      Module      = "audit"
    }
  )
}

# ========================================
# CloudTrail Resources
# ========================================

# S3 bucket for CloudTrail logs
resource "aws_s3_bucket" "cloudtrail" {
  count = var.enable_cloudtrail ? 1 : 0
  
  bucket        = "${local.name_prefix}-cloudtrail-${local.account_id}"
  force_destroy = var.environment != "production"

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-cloudtrail"
    }
  )
}

resource "aws_s3_bucket_versioning" "cloudtrail" {
  count = var.enable_cloudtrail ? 1 : 0
  
  bucket = aws_s3_bucket.cloudtrail[0].id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "cloudtrail" {
  count = var.enable_cloudtrail ? 1 : 0
  
  bucket = aws_s3_bucket.cloudtrail[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "cloudtrail" {
  count = var.enable_cloudtrail ? 1 : 0
  
  bucket = aws_s3_bucket.cloudtrail[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "cloudtrail" {
  count = var.enable_cloudtrail ? 1 : 0
  
  bucket = aws_s3_bucket.cloudtrail[0].id

  rule {
    id     = "retention-policy"
    status = "Enabled"

    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    transition {
      days          = 365
      storage_class = "DEEP_ARCHIVE"
    }

    expiration {
      days = var.cloudtrail_s3_retention_days
    }
  }
}

# S3 bucket policy for CloudTrail
resource "aws_s3_bucket_policy" "cloudtrail" {
  count = var.enable_cloudtrail ? 1 : 0
  
  bucket = aws_s3_bucket.cloudtrail[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AWSCloudTrailAclCheck"
        Effect = "Allow"
        Principal = {
          Service = "cloudtrail.amazonaws.com"
        }
        Action   = "s3:GetBucketAcl"
        Resource = aws_s3_bucket.cloudtrail[0].arn
      },
      {
        Sid    = "AWSCloudTrailWrite"
        Effect = "Allow"
        Principal = {
          Service = "cloudtrail.amazonaws.com"
        }
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.cloudtrail[0].arn}/*"
        Condition = {
          StringEquals = {
            "s3:x-amz-acl" = "bucket-owner-full-control"
          }
        }
      }
    ]
  })
}

# CloudWatch Log Group for CloudTrail
resource "aws_cloudwatch_log_group" "cloudtrail" {
  count = var.enable_cloudtrail ? 1 : 0
  
  name              = "/aws/cloudtrail/${local.name_prefix}"
  retention_in_days = var.cloudtrail_retention_days

  tags = merge(
    local.common_tags,
    {
      Name = "/aws/cloudtrail/${local.name_prefix}"
    }
  )
}

# IAM role for CloudTrail to CloudWatch Logs
resource "aws_iam_role" "cloudtrail_cloudwatch" {
  count = var.enable_cloudtrail ? 1 : 0
  
  name = "${local.name_prefix}-cloudtrail-cloudwatch"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "cloudtrail.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "cloudtrail_cloudwatch" {
  count = var.enable_cloudtrail ? 1 : 0
  
  name = "cloudtrail-cloudwatch-logs"
  role = aws_iam_role.cloudtrail_cloudwatch[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AWSCloudTrailCreateLogStream"
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "${aws_cloudwatch_log_group.cloudtrail[0].arn}:*"
      }
    ]
  })
}

# CloudTrail
resource "aws_cloudtrail" "main" {
  count = var.enable_cloudtrail ? 1 : 0
  
  name                          = "${local.name_prefix}-trail"
  s3_bucket_name                = aws_s3_bucket.cloudtrail[0].id
  include_global_service_events = true
  is_multi_region_trail         = true
  enable_log_file_validation    = true
  
  cloud_watch_logs_group_arn = "${aws_cloudwatch_log_group.cloudtrail[0].arn}:*"
  cloud_watch_logs_role_arn  = aws_iam_role.cloudtrail_cloudwatch[0].arn

  # Event selectors for data events (optional)
  dynamic "event_selector" {
    for_each = var.enable_s3_data_events ? [1] : []
    content {
      read_write_type           = "All"
      include_management_events = true

      data_resource {
        type   = "AWS::S3::Object"
        values = ["arn:${data.aws_partition.current.partition}:s3:::*/*"]
      }
    }
  }

  # Insights for anomaly detection (additional cost)
  dynamic "insight_selector" {
    for_each = var.enable_cloudtrail_insights ? [1] : []
    content {
      insight_type = "ApiCallRateInsight"
    }
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-cloudtrail"
    }
  )

  depends_on = [
    aws_s3_bucket_policy.cloudtrail,
    aws_iam_role_policy.cloudtrail_cloudwatch
  ]
}

# ========================================
# GuardDuty Resources
# ========================================

# GuardDuty detector
resource "aws_guardduty_detector" "main" {
  count = var.enable_guardduty ? 1 : 0
  
  enable                       = true
  finding_publishing_frequency = var.guardduty_finding_frequency

  datasources {
    s3_logs {
      enable = var.enable_guardduty_s3_protection
    }
    
    kubernetes {
      audit_logs {
        enable = var.enable_guardduty_eks_protection
      }
    }
    
    malware_protection {
      scan_ec2_instance_with_findings {
        ebs_volumes {
          enable = var.enable_guardduty_malware_protection
        }
      }
    }
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-guardduty"
    }
  )
}

# S3 bucket for GuardDuty findings export (optional)
resource "aws_s3_bucket" "guardduty_findings" {
  count = var.enable_guardduty && var.enable_guardduty_export ? 1 : 0
  
  bucket        = "${local.name_prefix}-guardduty-findings-${local.account_id}"
  force_destroy = var.environment != "production"

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-guardduty-findings"
    }
  )
}

resource "aws_s3_bucket_public_access_block" "guardduty_findings" {
  count = var.enable_guardduty && var.enable_guardduty_export ? 1 : 0
  
  bucket = aws_s3_bucket.guardduty_findings[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# S3 bucket policy to allow GuardDuty to write findings
resource "aws_s3_bucket_policy" "guardduty_findings" {
  count = var.enable_guardduty && var.enable_guardduty_export ? 1 : 0
  
  bucket = aws_s3_bucket.guardduty_findings[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowGuardDutyAccess"
        Effect = "Allow"
        Principal = {
          Service = "guardduty.amazonaws.com"
        }
        Action = [
          "s3:PutObject"
        ]
        Resource = "${aws_s3_bucket.guardduty_findings[0].arn}/*"
      },
      {
        Sid    = "AllowGuardDutyGetBucketLocation"
        Effect = "Allow"
        Principal = {
          Service = "guardduty.amazonaws.com"
        }
        Action = [
          "s3:GetBucketLocation"
        ]
        Resource = aws_s3_bucket.guardduty_findings[0].arn
      }
    ]
  })

  depends_on = [aws_s3_bucket_public_access_block.guardduty_findings]
}

resource "aws_guardduty_publishing_destination" "findings" {
  count = var.enable_guardduty && var.enable_guardduty_export ? 1 : 0
  
  detector_id     = aws_guardduty_detector.main[0].id
  destination_arn = aws_s3_bucket.guardduty_findings[0].arn
  kms_key_arn     = null  # Use default encryption

  depends_on = [aws_s3_bucket_policy.guardduty_findings]
}

# ========================================
# Security Alerts
# ========================================

# SNS topic for security alerts
resource "aws_sns_topic" "security_alerts" {
  name = "${local.name_prefix}-security-alerts"

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-security-alerts"
    }
  )
}

# SNS topic subscription for email
resource "aws_sns_topic_subscription" "security_alerts_email" {
  topic_arn = aws_sns_topic.security_alerts.arn
  protocol  = "email"
  endpoint  = var.alert_email
}

# EventBridge rule for GuardDuty findings
resource "aws_cloudwatch_event_rule" "guardduty_findings" {
  count = var.enable_guardduty ? 1 : 0
  
  name        = "${local.name_prefix}-guardduty-findings"
  description = "Capture GuardDuty findings with severity >= 4"

  event_pattern = jsonencode({
    source      = ["aws.guardduty"]
    detail-type = ["GuardDuty Finding"]
    detail = {
      severity = [
        { numeric = [">=", 4] }
      ]
    }
  })

  tags = local.common_tags
}

resource "aws_cloudwatch_event_target" "guardduty_to_sns" {
  count = var.enable_guardduty ? 1 : 0
  
  rule      = aws_cloudwatch_event_rule.guardduty_findings[0].name
  target_id = "SendToSNS"
  arn       = aws_sns_topic.security_alerts.arn
}

# SNS topic policy to allow EventBridge
resource "aws_sns_topic_policy" "security_alerts" {
  arn = aws_sns_topic.security_alerts.arn

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "events.amazonaws.com"
        }
        Action   = "SNS:Publish"
        Resource = aws_sns_topic.security_alerts.arn
      }
    ]
  })
}

# ========================================
# CloudWatch Alarms for CloudTrail Metrics
# ========================================

# Metric filter for unauthorized API calls
resource "aws_cloudwatch_log_metric_filter" "unauthorized_api_calls" {
  count = var.enable_cloudtrail ? 1 : 0
  
  name           = "${local.name_prefix}-unauthorized-api-calls"
  log_group_name = aws_cloudwatch_log_group.cloudtrail[0].name
  pattern        = "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") }"

  metric_transformation {
    name      = "UnauthorizedAPICalls"
    namespace = "CloudTrail/${local.name_prefix}"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "unauthorized_api_calls" {
  count = var.enable_cloudtrail ? 1 : 0
  
  alarm_name          = "${local.name_prefix}-unauthorized-api-calls"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "UnauthorizedAPICalls"
  namespace           = "CloudTrail/${local.name_prefix}"
  period              = 300
  statistic           = "Sum"
  threshold           = 5
  alarm_description   = "Alerts on multiple unauthorized API calls"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  treat_missing_data = "notBreaching"
}

# Metric filter for root account usage
resource "aws_cloudwatch_log_metric_filter" "root_usage" {
  count = var.enable_cloudtrail ? 1 : 0
  
  name           = "${local.name_prefix}-root-account-usage"
  log_group_name = aws_cloudwatch_log_group.cloudtrail[0].name
  pattern        = "{ $.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS && $.eventType != \"AwsServiceEvent\" }"

  metric_transformation {
    name      = "RootAccountUsage"
    namespace = "CloudTrail/${local.name_prefix}"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "root_usage" {
  count = var.enable_cloudtrail ? 1 : 0
  
  alarm_name          = "${local.name_prefix}-root-account-usage"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "RootAccountUsage"
  namespace           = "CloudTrail/${local.name_prefix}"
  period              = 60
  statistic           = "Sum"
  threshold           = 1
  alarm_description   = "Alerts when root account is used"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  treat_missing_data = "notBreaching"
}

# Metric filter for IAM policy changes
resource "aws_cloudwatch_log_metric_filter" "iam_policy_changes" {
  count = var.enable_cloudtrail ? 1 : 0
  
  name           = "${local.name_prefix}-iam-policy-changes"
  log_group_name = aws_cloudwatch_log_group.cloudtrail[0].name
  pattern        = "{($.eventName=DeleteGroupPolicy)||($.eventName=DeleteRolePolicy)||($.eventName=DeleteUserPolicy)||($.eventName=PutGroupPolicy)||($.eventName=PutRolePolicy)||($.eventName=PutUserPolicy)||($.eventName=CreatePolicy)||($.eventName=DeletePolicy)||($.eventName=CreatePolicyVersion)||($.eventName=DeletePolicyVersion)||($.eventName=AttachRolePolicy)||($.eventName=DetachRolePolicy)||($.eventName=AttachUserPolicy)||($.eventName=DetachUserPolicy)||($.eventName=AttachGroupPolicy)||($.eventName=DetachGroupPolicy)}"

  metric_transformation {
    name      = "IAMPolicyChanges"
    namespace = "CloudTrail/${local.name_prefix}"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "iam_policy_changes" {
  count = var.enable_cloudtrail ? 1 : 0
  
  alarm_name          = "${local.name_prefix}-iam-policy-changes"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "IAMPolicyChanges"
  namespace           = "CloudTrail/${local.name_prefix}"
  period              = 300
  statistic           = "Sum"
  threshold           = 1
  alarm_description   = "Alerts on IAM policy changes"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  treat_missing_data = "notBreaching"
}
