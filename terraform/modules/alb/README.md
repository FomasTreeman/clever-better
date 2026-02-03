# ALB Module

This module creates an Application Load Balancer for the ML service with HTTPS support, health checks, and WAF integration.

## Features

- **Internet-facing ALB** for external access
- **HTTPS with TLS 1.3** using ACM certificates
- **HTTP to HTTPS redirect** for secure access
- **Dual protocol support**: HTTP REST API and gRPC
- **Health checks** with configurable parameters
- **Session stickiness** (optional)
- **Access logs** to S3 (optional)
- **WAF integration** for DDoS and application protection
- **Cross-zone load balancing** for high availability

## Architecture

```
                    Internet
                       │
                       ▼
              ┌────────────────┐
              │   Route 53     │
              │  (DNS/CNAME)   │
              └────────┬───────┘
                       │
                       ▼
┌──────────────────────────────────────────┐
│     Application Load Balancer            │
│                                          │
│  ┌────────────┐         ┌─────────────┐ │
│  │  HTTP:80   │────────►│ HTTPS:443   │ │
│  │ (Redirect) │         │ (TLS 1.3)   │ │
│  └────────────┘         └──────┬──────┘ │
│                                │        │
│         ┌──────────────────────┴─────┐  │
│         ▼                            ▼  │
│  ┌─────────────┐            ┌──────────────┐
│  │  HTTP:8000  │            │  gRPC:50051  │
│  │ Target Group│            │ Target Group │
│  └─────────────┘            └──────────────┘
└──────────────────────────────────────────┘
         │                            │
         └────────────┬───────────────┘
                      ▼
              ┌───────────────┐
              │  ECS Tasks    │
              │  (Fargate)    │
              └───────────────┘
```

## Usage

```hcl
module "alb" {
  source = "../../modules/alb"

  environment = "production"
  project_name = "clever-better"
  
  # Network configuration
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.public_subnet_ids
  security_group_ids  = [module.security.alb_security_group_id]
  
  # SSL/TLS configuration
  certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/xxxxx"
  
  # Protection
  enable_deletion_protection = true
  waf_web_acl_arn           = module.waf.web_acl_arn
  
  # Optional features
  enable_access_logs = true
  enable_stickiness  = false
  
  # Health checks
  health_check_path     = "/health"
  health_check_interval = 30
  health_check_timeout  = 5
  
  tags = {
    Project   = "clever-better"
    ManagedBy = "terraform"
  }
}
```

## SSL/TLS Certificate

### Prerequisites

You must create an ACM certificate **before** deploying this module:

```bash
# Request certificate
aws acm request-certificate \
  --domain-name api.clever-better.com \
  --validation-method DNS \
  --region us-east-1

# Validate via DNS (add CNAME record to your DNS)
# Or use email validation

# Get certificate ARN after validation
aws acm list-certificates --region us-east-1
```

### Certificate Requirements

- **Region**: Must be in the same region as the ALB
- **Status**: Must be "Issued" (validated)
- **Domain**: Should match your intended domain name
- **Renewal**: ACM automatically renews certificates

## Target Groups

### HTTP Target Group

- **Port**: 8000
- **Protocol**: HTTP
- **Use case**: REST API endpoints
- **Default action**: All HTTPS traffic goes here unless matched by gRPC rule

### gRPC Target Group

- **Port**: 50051
- **Protocol**: HTTP/2 (GRPC)
- **Use case**: gRPC service calls
- **Routing**: Traffic with `content-type: application/grpc*` or path `/ml.MLService/*`

### Health Check Configuration

Both target groups use the same health check settings:

```hcl
health_check {
  enabled             = true
  healthy_threshold   = 2     # 2 consecutive successes
  unhealthy_threshold = 3     # 3 consecutive failures
  timeout             = 5     # 5 seconds
  interval            = 30    # Check every 30 seconds
  path                = "/health"
  protocol            = "HTTP"
  matcher             = "200" # HTTP 200 OK
}
```

## Listener Rules

### HTTP Listener (Port 80)

- **Action**: Redirect to HTTPS (301 permanent redirect)
- **Purpose**: Force all traffic to use encryption

### HTTPS Listener (Port 443)

- **SSL Policy**: `ELBSecurityPolicy-TLS13-1-2-2021-06`
  - TLS 1.3 preferred
  - TLS 1.2 supported for compatibility
  - Strong cipher suites only
- **Default Action**: Forward to HTTP target group
- **Rules**:
  - Priority 100: Forward gRPC traffic to gRPC target group

## Access Logs

When enabled, ALB access logs are stored in S3:

### Log Format

```
https 2024-01-01T12:00:00.000000Z app/clever-better-production-alb/xxx 
192.0.2.1:12345 10.0.1.5:8000 0.001 0.002 0.000 200 200 
512 256 "GET https://api.clever-better.com:443/health HTTP/2.0" 
"Mozilla/5.0..." - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/...
```

### Log Lifecycle

- **Retention**: 90 days
- **Transition**: Move to Standard-IA after 30 days
- **Cost**: ~$0.005 per GB stored + ~$0.005 per GB transferred

### Querying Logs

```bash
# Download logs
aws s3 sync s3://<bucket-name>/AWSLogs/<account-id>/ ./alb-logs/

# Analyze with athena or grep
grep "error" alb-logs/*.log.gz | zcat
```

## WAF Integration

When `waf_web_acl_arn` is provided, the ALB is protected by AWS WAF:

- **DDoS protection**: Rate limiting and IP blocking
- **SQL injection protection**: Detects SQL injection attempts
- **XSS protection**: Blocks cross-site scripting
- **Geographic blocking**: Restrict traffic by country (if configured)
- **Bot protection**: Identify and block malicious bots

See the WAF module documentation for configuration details.

## DNS Configuration

### Using Route 53

```hcl
resource "aws_route53_record" "api" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "api.clever-better.com"
  type    = "A"

  alias {
    name                   = module.alb.alb_dns_name
    zone_id                = module.alb.alb_zone_id
    evaluate_target_health = true
  }
}
```

### Using External DNS

Create a CNAME record pointing to the ALB DNS name:

```
api.clever-better.com  CNAME  clever-better-production-alb-123456789.us-east-1.elb.amazonaws.com
```

## Monitoring

### CloudWatch Metrics

The ALB automatically publishes metrics to CloudWatch:

- **RequestCount**: Number of requests
- **TargetResponseTime**: Response time from targets
- **HTTPCode_Target_2XX_Count**: Successful responses
- **HTTPCode_Target_4XX_Count**: Client errors
- **HTTPCode_Target_5XX_Count**: Server errors
- **HTTPCode_ELB_5XX_Count**: ELB errors
- **TargetConnectionErrorCount**: Connection failures
- **UnHealthyHostCount**: Number of unhealthy targets
- **HealthyHostCount**: Number of healthy targets

### Recommended Alarms

```hcl
# High 5xx error rate
resource "aws_cloudwatch_metric_alarm" "alb_5xx_high" {
  alarm_name          = "alb-5xx-errors-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "HTTPCode_Target_5XX_Count"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Sum"
  threshold           = 10
  alarm_description   = "ALB is receiving high 5xx errors from targets"

  dimensions = {
    LoadBalancer = module.alb.alb_arn
  }
}

# Unhealthy targets
resource "aws_cloudwatch_metric_alarm" "alb_unhealthy_targets" {
  alarm_name          = "alb-unhealthy-targets"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "UnHealthyHostCount"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Maximum"
  threshold           = 0
  alarm_description   = "ALB has unhealthy targets"

  dimensions = {
    LoadBalancer = module.alb.alb_arn
    TargetGroup  = module.alb.ml_http_target_group_arn
  }
}

# High response time
resource "aws_cloudwatch_metric_alarm" "alb_response_time" {
  alarm_name          = "alb-response-time-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "TargetResponseTime"
  namespace           = "AWS/ApplicationELB"
  period              = 300
  statistic           = "Average"
  threshold           = 1.0  # 1 second
  alarm_description   = "ALB target response time is high"

  dimensions = {
    LoadBalancer = module.alb.alb_arn
  }
}
```

## Security Best Practices

1. **Use HTTPS only**: Redirect HTTP to HTTPS
2. **Strong TLS policy**: Use TLS 1.3/1.2 with strong ciphers
3. **WAF protection**: Enable WAF for DDoS and application attacks
4. **Security groups**: Restrict inbound to 80/443, outbound to target ports
5. **Access logs**: Enable for audit and troubleshooting
6. **Deletion protection**: Enable in production
7. **Certificate management**: Use ACM for automatic renewal

## Cost Optimization

### Development/Staging

```hcl
enable_deletion_protection = false
enable_access_logs         = false  # Save on S3 costs
```

**Estimated Cost**: ~$20-25/month

### Production

```hcl
enable_deletion_protection = true
enable_access_logs         = true
waf_web_acl_arn           = module.waf.web_acl_arn  # +$5/month
```

**Estimated Cost**: ~$25-30/month + WAF costs

### Cost Breakdown

- **ALB hours**: ~$16.20/month (24/7)
- **LCU (Load Balancer Capacity Units)**: ~$5-10/month (depends on traffic)
- **Access logs**: ~$1-5/month (depends on traffic)
- **Data transfer**: $0.09/GB outbound

## Troubleshooting

### Health Check Failures

1. Check target security group allows ALB security group on target port
2. Verify target is listening on the correct port
3. Ensure health check path returns 200 OK
4. Check CloudWatch Logs for application errors
5. Verify VPC routing (private subnets can reach targets)

### SSL/TLS Errors

1. Verify certificate is in "Issued" status
2. Ensure certificate covers the domain name
3. Check certificate is in the correct region
4. Validate DNS points to ALB
5. Test with `curl -v https://domain.com`

### High Latency

1. Check target response time metrics
2. Review application performance
3. Consider increasing target task count
4. Enable auto-scaling for targets
5. Check for unhealthy targets causing retries

### 503 Service Unavailable

1. All targets are unhealthy - check health checks
2. No registered targets - check ECS service
3. Target group has no capacity - scale up tasks

## Inputs

| Name | Type | Default | Description |
|------|------|---------|-------------|
| environment | string | required | Environment name (dev, staging, production) |
| project_name | string | "clever-better" | Project name for resource naming |
| vpc_id | string | required | VPC ID |
| subnet_ids | list(string) | required | Public subnet IDs (minimum 2) |
| security_group_ids | list(string) | required | Security group IDs for ALB |
| certificate_arn | string | required | ACM certificate ARN for HTTPS |
| enable_deletion_protection | bool | true | Enable deletion protection |
| enable_access_logs | bool | false | Enable access logs to S3 |
| enable_stickiness | bool | false | Enable session stickiness |
| health_check_path | string | "/health" | Health check path |
| health_check_interval | number | 30 | Health check interval (seconds) |
| health_check_timeout | number | 5 | Health check timeout (seconds) |
| deregistration_delay | number | 30 | Target deregistration delay (seconds) |
| waf_web_acl_arn | string | "" | WAF Web ACL ARN (optional) |
| tags | map(string) | {} | Additional resource tags |

## Outputs

| Name | Description |
|------|-------------|
| alb_id | ALB ID |
| alb_arn | ALB ARN |
| alb_dns_name | ALB DNS name (for CNAME/A record) |
| alb_zone_id | ALB Route53 zone ID (for alias record) |
| http_listener_arn | HTTP listener ARN |
| https_listener_arn | HTTPS listener ARN |
| ml_http_target_group_arn | ML HTTP target group ARN |
| ml_http_target_group_name | ML HTTP target group name |
| ml_grpc_target_group_arn | ML gRPC target group ARN |
| ml_grpc_target_group_name | ML gRPC target group name |
| access_logs_bucket_name | Access logs S3 bucket name |

## References

- [Application Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/)
- [HTTPS Listeners](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/create-https-listener.html)
- [gRPC on ALB](https://aws.amazon.com/blogs/aws/new-application-load-balancer-support-for-end-to-end-http-2-and-grpc/)
- [ACM Certificates](https://docs.aws.amazon.com/acm/latest/userguide/)
- [ALB Access Logs](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-access-logs.html)
