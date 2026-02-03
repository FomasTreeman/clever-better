# Terraform Security Overview

## Defense-in-Depth
- Network isolation (public vs private subnets)
- Security group least-privilege
- WAF protections for ingress traffic
- IAM policies scoped by resource prefix
- Secrets Manager for sensitive data

## Network Controls
- Private app and data subnets
- NAT gateways for outbound access
- VPC Flow Logs enabled with retention

## Application Controls
- WAF managed rule sets (IP reputation, common rules, bad inputs)
- Rate limiting and optional geo blocking

## Data Controls
- Secrets Manager with rotation (optional)
- KMS encryption for secrets
- Encrypted buckets for logs

## Monitoring & Response
- CloudTrail and GuardDuty
- CloudWatch alarms for security events
- SNS for notifications

## Checklist
- [ ] Backend configured with encryption and locking
- [ ] WAF logging enabled in staging/prod
- [ ] Secrets rotation enabled for production
- [ ] IAM policies reviewed for least privilege
