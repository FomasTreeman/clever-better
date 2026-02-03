# IAM Module

Creates IAM roles and policies for ECS tasks, RDS monitoring, VPC Flow Logs, and CloudWatch Events.

## Highlights
- ECS task execution role
- Bot task role
- ML service task role
- Secrets Manager read policy
- CloudWatch Logs write policy
- CloudWatch Metrics write policy

## Least Privilege
Policies are scoped to named prefixes for secrets and log groups.
