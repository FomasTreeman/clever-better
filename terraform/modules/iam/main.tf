locals {
  tags = merge({
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }, var.tags)
}

data "aws_iam_policy_document" "ecs_task_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ecs_execution" {
  name               = "${var.project_name}-${var.environment}-ecs-exec"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
  tags               = merge(local.tags, { Component = "ecs-execution" })
}

resource "aws_iam_role_policy_attachment" "ecs_exec_managed" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

data "aws_iam_policy_document" "secrets_read" {
  statement {
    actions   = ["secretsmanager:GetSecretValue", "secretsmanager:DescribeSecret"]
    resources = ["arn:aws:secretsmanager:*:*:secret:${var.secrets_prefix}/*"]
  }
}

resource "aws_iam_policy" "secrets_read" {
  name   = "${var.project_name}-${var.environment}-secrets-read"
  policy = data.aws_iam_policy_document.secrets_read.json
}

resource "aws_iam_role_policy_attachment" "ecs_exec_secrets" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = aws_iam_policy.secrets_read.arn
}

data "aws_iam_policy_document" "logs_write" {
  statement {
    actions   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["arn:aws:logs:*:*:log-group:${var.log_group_prefix}*"]
  }
}

resource "aws_iam_policy" "logs_write" {
  name   = "${var.project_name}-${var.environment}-logs-write"
  policy = data.aws_iam_policy_document.logs_write.json
}

resource "aws_iam_role_policy_attachment" "ecs_exec_logs" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = aws_iam_policy.logs_write.arn
}

resource "aws_iam_role" "bot_task" {
  name               = "${var.project_name}-${var.environment}-bot-task"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
  tags               = merge(local.tags, { Component = "bot" })
}

resource "aws_iam_role_policy_attachment" "bot_secrets" {
  role       = aws_iam_role.bot_task.name
  policy_arn = aws_iam_policy.secrets_read.arn
}

resource "aws_iam_role_policy_attachment" "bot_logs" {
  role       = aws_iam_role.bot_task.name
  policy_arn = aws_iam_policy.logs_write.arn
}

resource "aws_iam_policy" "metrics_write" {
  name = "${var.project_name}-${var.environment}-metrics-write"
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect   = "Allow",
      Action   = ["cloudwatch:PutMetricData"],
      Resource = "*"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "bot_metrics" {
  role       = aws_iam_role.bot_task.name
  policy_arn = aws_iam_policy.metrics_write.arn
}

resource "aws_iam_role" "ml_task" {
  name               = "${var.project_name}-${var.environment}-ml-task"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
  tags               = merge(local.tags, { Component = "ml-service" })
}

resource "aws_iam_role_policy_attachment" "ml_secrets" {
  role       = aws_iam_role.ml_task.name
  policy_arn = aws_iam_policy.secrets_read.arn
}

resource "aws_iam_role_policy_attachment" "ml_logs" {
  role       = aws_iam_role.ml_task.name
  policy_arn = aws_iam_policy.logs_write.arn
}

resource "aws_iam_role_policy_attachment" "ml_metrics" {
  role       = aws_iam_role.ml_task.name
  policy_arn = aws_iam_policy.metrics_write.arn
}

resource "aws_iam_role" "rds_monitoring" {
  name = "${var.project_name}-${var.environment}-rds-monitoring"
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect    = "Allow",
      Action    = "sts:AssumeRole",
      Principal = { Service = "monitoring.rds.amazonaws.com" }
    }]
  })
  tags = merge(local.tags, { Component = "rds-monitoring" })
}

resource "aws_iam_role_policy_attachment" "rds_monitoring" {
  role       = aws_iam_role.rds_monitoring.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
}

resource "aws_iam_role" "vpc_flow_logs" {
  name = "${var.project_name}-${var.environment}-vpc-flow-logs"
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect    = "Allow",
      Action    = "sts:AssumeRole",
      Principal = { Service = "vpc-flow-logs.amazonaws.com" }
    }]
  })
  tags = merge(local.tags, { Component = "vpc-flow-logs" })
}

resource "aws_iam_role_policy" "vpc_flow_logs" {
  name = "${var.project_name}-${var.environment}-vpc-flow-logs"
  role = aws_iam_role.vpc_flow_logs.id
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect   = "Allow",
      Action   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"],
      Resource = "*"
    }]
  })
}

resource "aws_iam_role" "cloudwatch_events" {
  name = "${var.project_name}-${var.environment}-events"
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect    = "Allow",
      Action    = "sts:AssumeRole",
      Principal = { Service = "events.amazonaws.com" }
    }]
  })
  tags = merge(local.tags, { Component = "cloudwatch-events" })
}

resource "aws_iam_role_policy" "cloudwatch_events" {
  name = "${var.project_name}-${var.environment}-events"
  role = aws_iam_role.cloudwatch_events.id
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect   = "Allow",
      Action   = ["ecs:RunTask", "iam:PassRole"],
      Resource = "*"
    }]
  })
}
