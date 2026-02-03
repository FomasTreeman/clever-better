# Cost Estimation

Estimated monthly costs are approximate and depend on traffic, log volume, and region (us-east-1). These estimates assume 24/7 operation.

## Production (Estimated)
- RDS db.r6g.large Multi-AZ: ~$350/month
- RDS storage (100 GB): ~$23/month
- RDS backups (30 days): ~$10/month
- ECS Fargate (bot + ML): ~$150/month
- ALB: ~$25/month
- NAT Gateway (2x): ~$70/month
- CloudTrail: ~$5/month
- GuardDuty: ~$5/month
- Total estimated: **~$638/month**

## Development (Estimated)
- RDS db.t4g.medium (Single-AZ): ~$50/month
- ECS Fargate (minimal tasks): ~$30/month
- ALB: ~$25/month
- NAT Gateway (1x): ~$35/month
- CloudTrail/GuardDuty: ~$5-10/month
- Total estimated: **~$145-$150/month**

## Staging (Estimated)
- RDS db.r6g.large (Single-AZ or Multi-AZ): ~$200-$350/month
- ECS Fargate: ~$80-$100/month
- ALB: ~$25/month
- NAT Gateway (2x): ~$70/month
- CloudTrail/GuardDuty: ~$10/month
- Total estimated: **~$385-$555/month**

## Cost Optimization Tips
- Use db.t4g instances for dev/staging
- Single AZ for non-production environments
- Disable CloudTrail insights in dev
- Use Fargate Spot for non-critical workloads
- Consider Reserved Instances for RDS in production
