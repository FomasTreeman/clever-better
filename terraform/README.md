# Terraform Infrastructure

This directory contains the foundational AWS infrastructure for Clever Better. The focus is on networking, security, IAM, and secrets management to support later compute and data tiers.

## Module Overview
- VPC: Network foundation (public, private app, private data subnets)
- Security: Security groups for ALB, ECS, RDS, VPC endpoints
- WAF: Web ACL, rate limiting, managed rule sets, logging
- IAM: Roles and least-privilege policies for ECS and services
- Secrets: Secrets Manager and KMS for credentials
- Monitoring: CloudTrail and GuardDuty baseline

## Environments
Each environment has its own configuration:
- terraform/environments/dev
- terraform/environments/staging
- terraform/environments/production

## Backend Initialization
1) Create backend resources with scripts:
   - terraform/scripts/setup-backend.sh
2) Copy backend.tf.example to backend.tf
3) Run terraform init in the environment directory

## Deployment Workflow
1) Configure terraform.tfvars (from terraform.tfvars.example)
2) terraform init
3) terraform validate
4) terraform plan
5) terraform apply

## Variable Conventions
- Use environment-specific tfvars
- Prefer explicit naming and documented defaults
- All modules accept tags map for standardization

## Security Considerations
- Least-privilege IAM roles
- WAF protections with managed rule sets
- Secrets Manager for credentials
- VPC Flow Logs and CloudTrail for audit

## Module Dependency Graph
```mermaid
graph TB
  VPC[VPC Module] --> SG[Security Module]
  SG --> WAF[WAF Module]
  IAM[IAM Module] --> Secrets[Secrets Module]
  Monitoring[Monitoring Module] --> WAF
```
