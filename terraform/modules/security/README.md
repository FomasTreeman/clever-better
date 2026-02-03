# Security Groups Module

Defines security groups for ALB, application, database, and VPC endpoints.

## Traffic Flow
- ALB -> App on 8000/50051/8080
- App -> DB on 5432
- App -> Internet on 443

## Least Privilege
Rules are scoped to specific ports and security group sources.
