# Handoff Notes

## Outputs Required for Next Phase
- VPC ID
- Public subnet IDs
- Private app subnet IDs
- Private data subnet IDs
- ALB security group ID
- Application security group ID
- Database security group ID
- WAF Web ACL ARN
- ECS task role ARNs

## Integration Points
- ECS module: requires VPC, app subnets, security group IDs
- RDS module: requires data subnets, database SG
- ALB module: requires public subnets, ALB SG, WAF ACL ARN

## Notes
- Modules are designed for least-privilege
- Adjust WAF rules and IP allowlists as needed
