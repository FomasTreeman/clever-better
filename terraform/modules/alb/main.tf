# ALB Module - Application Load Balancer
# Provides internet-facing ALB with HTTPS/HTTP listeners and target groups

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Data source for current AWS account
data "aws_caller_identity" "current" {}

# Data source for ELB service account (for S3 access logs)
data "aws_elb_service_account" "main" {}

# Local variables
locals {
  name_prefix = "${var.project_name}-${var.environment}"
  
  common_tags = merge(
    var.tags,
    {
      Name        = "${local.name_prefix}-alb"
      Environment = var.environment
      ManagedBy   = "terraform"
      Module      = "alb"
    }
  )
}

# S3 bucket for ALB access logs (optional)
resource "aws_s3_bucket" "access_logs" {
  count = var.enable_access_logs && var.access_logs_bucket == "" ? 1 : 0
  
  bucket = "${local.name_prefix}-alb-logs-${data.aws_caller_identity.current.account_id}"
  
  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-alb-logs"
    }
  )
}

resource "aws_s3_bucket_public_access_block" "access_logs" {
  count = var.enable_access_logs && var.access_logs_bucket == "" ? 1 : 0
  
  bucket = aws_s3_bucket.access_logs[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "access_logs" {
  count = var.enable_access_logs && var.access_logs_bucket == "" ? 1 : 0
  
  bucket = aws_s3_bucket.access_logs[0].id

  rule {
    id     = "delete-old-logs"
    status = "Enabled"

    expiration {
      days = 90
    }
  }

  rule {
    id     = "transition-to-ia"
    status = "Enabled"

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
  }
}

resource "aws_s3_bucket_policy" "access_logs" {
  count = var.enable_access_logs && var.access_logs_bucket == "" ? 1 : 0
  
  bucket = aws_s3_bucket.access_logs[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          AWS = data.aws_elb_service_account.main.arn
        }
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.access_logs[0].arn}/*"
      },
      {
        Effect = "Allow"
        Principal = {
          Service = "logdelivery.elasticloadbalancing.amazonaws.com"
        }
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.access_logs[0].arn}/*"
      }
    ]
  })
}

# Application Load Balancer
resource "aws_lb" "this" {
  name               = "${local.name_prefix}-alb"
  load_balancer_type = "application"
  internal           = false
  
  # Network configuration
  subnets         = var.subnet_ids
  security_groups = var.security_group_ids
  
  # Configuration
  enable_deletion_protection       = var.enable_deletion_protection
  enable_http2                     = true
  enable_cross_zone_load_balancing = true
  
  # Access logs
  dynamic "access_logs" {
    for_each = var.enable_access_logs ? [1] : []
    content {
      bucket  = var.access_logs_bucket != "" ? var.access_logs_bucket : aws_s3_bucket.access_logs[0].id
      enabled = true
    }
  }
  
  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-alb"
    }
  )
}

# Target Group for ML Service HTTP
resource "aws_lb_target_group" "ml_http" {
  name        = "${local.name_prefix}-ml-http"
  port        = 8000
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "ip"  # Required for Fargate

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = var.health_check_timeout
    interval            = var.health_check_interval
    path                = var.health_check_path
    protocol            = "HTTP"
    matcher             = "200"
  }

  deregistration_delay = var.deregistration_delay

  dynamic "stickiness" {
    for_each = var.enable_stickiness ? [1] : []
    content {
      type            = "lb_cookie"
      enabled         = true
      cookie_duration = 86400  # 24 hours
    }
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-ml-http-tg"
    }
  )
}

# Target Group for ML Service gRPC
resource "aws_lb_target_group" "ml_grpc" {
  name                 = "${local.name_prefix}-ml-grpc"
  port                 = 50051
  protocol             = "HTTP"
  protocol_version     = "GRPC"
  vpc_id               = var.vpc_id
  target_type          = "ip"  # Required for Fargate

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = var.health_check_timeout
    interval            = var.health_check_interval
    path                = var.health_check_path
    protocol            = "HTTP"
    matcher             = "0-99"  # gRPC status codes
  }

  deregistration_delay = var.deregistration_delay

  dynamic "stickiness" {
    for_each = var.enable_stickiness ? [1] : []
    content {
      type            = "lb_cookie"
      enabled         = true
      cookie_duration = 86400  # 24 hours
    }
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-ml-grpc-tg"
    }
  )
}

# HTTP Listener (redirect to HTTPS)
resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.this.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-http-listener"
    }
  )
}

# HTTPS Listener
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.this.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = var.certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ml_http.arn
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-https-listener"
    }
  )
}

# Listener Rule for gRPC traffic
resource "aws_lb_listener_rule" "grpc" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 100

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ml_grpc.arn
  }

  condition {
    path_pattern {
      values = ["/ml.MLService/*"]
    }
  }

  condition {
    http_header {
      http_header_name = "content-type"
      values           = ["application/grpc*"]
    }
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-grpc-rule"
    }
  )
}

# WAF Association (optional)
resource "aws_wafv2_web_acl_association" "alb" {
  count = var.waf_web_acl_arn != "" ? 1 : 0

  resource_arn = aws_lb.this.arn
  web_acl_arn  = var.waf_web_acl_arn
}
