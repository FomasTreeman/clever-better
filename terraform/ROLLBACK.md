# Rollback Procedures

## Rollback Using State History
- S3 backend must have versioning enabled
- Retrieve a prior state version and restore

## Emergency Rollback
1) Disable apply in CI/CD
2) Restore previous state file
3) Run terraform plan to confirm
4) terraform apply to reconcile

## Validation
- Verify critical resources (VPC, SGs, WAF, IAM)
- Confirm monitoring and alarms
