# Operational Runbook

This runbook provides comprehensive operational procedures for the Clever Better trading bot system. It serves as the primary reference for deployment, monitoring, incident response, and maintenance activities.

## Table of Contents

- [Deployment Procedures](#deployment-procedures)
- [Monitoring and Alerting](#monitoring-and-alerting)
- [Incident Response](#incident-response)
- [Troubleshooting Procedures](#troubleshooting-procedures)
- [Backup and Recovery](#backup-and-recovery)
- [Maintenance Tasks](#maintenance-tasks)
- [Access and Permissions](#access-and-permissions)
- [Runbook Maintenance](#runbook-maintenance)

## Deployment Procedures

### Pre-Deployment Checklist

Before deploying to any environment:

- [ ] All tests passing (unit, integration, E2E)
- [ ] Code review completed and approved
- [ ] Database migrations reviewed and tested
- [ ] Configuration validated for target environment
- [ ] Dependencies updated and security-scanned
- [ ] Backup of current production state completed
- [ ] Rollback plan documented and tested
- [ ] Stakeholders notified of deployment window
- [ ] On-call engineer identified

### Development Environment Deployment

```bash
# Navigate to project root
cd /path/to/clever-better

# Ensure tests pass
make test-all

# Deploy infrastructure
cd terraform/environments/dev
terraform init
terraform plan -out=tfplan
terraform apply tfplan

# Build and push Docker images
make docker-build
make docker-push-dev

# Deploy application
cd ../../../
make deploy-dev

# Verify deployment
make smoke-test-dev
```

**Verification Steps:**
1. Check ECS task status: `aws ecs list-tasks --cluster clever-better-dev`
2. Verify health endpoints: `curl https://dev-api.clever-better.com/health`
3. Check CloudWatch logs for errors
4. Verify database connectivity
5. Test ML service predictions

### Staging Environment Deployment

```bash
# Merge to staging branch
git checkout staging
git merge develop
git push origin staging

# Deploy via CI/CD
# GitHub Actions will automatically trigger deployment

# Manual deployment (if needed)
cd terraform/environments/staging
terraform init
terraform plan -out=tfplan
terraform apply tfplan

# Deploy application
make deploy-staging

# Run smoke tests
make smoke-test-staging
```

**Verification Steps:**
1. Verify all ECS services running
2. Check ALB target health
3. Test end-to-end workflow with test data
4. Verify ML model predictions
5. Check metrics in CloudWatch dashboards

### Production Environment Deployment

**CRITICAL:** Production deployments require approval and should follow blue-green deployment process.

```bash
# Create production release
git tag -a v1.0.0 -m "Production release v1.0.0"
git push origin v1.0.0

# Pre-deployment steps
1. Notify stakeholders of deployment window
2. Enable maintenance mode (optional)
3. Take database snapshot
4. Verify backup integrity

# Blue-Green Deployment Process

# Step 1: Deploy to green environment
cd terraform/environments/production
terraform apply -var="deployment_slot=green"

# Step 2: Verify green environment
make smoke-test-prod-green

# Step 3: Update ALB to route traffic to green
terraform apply -var="active_slot=green"

# Step 4: Monitor for 15 minutes
# Watch CloudWatch metrics, error rates, latency

# Step 5: If issues detected, rollback immediately
terraform apply -var="active_slot=blue"

# Step 6: If stable, decommission blue environment
terraform destroy -var="deployment_slot=blue" -target=aws_ecs_service.bot_blue
```

**Post-Deployment Verification:**
1. Monitor error rates in CloudWatch
2. Check trading bot placing bets correctly
3. Verify ML predictions within expected range
4. Monitor database performance
5. Check all alarms in CloudWatch
6. Review application logs for errors
7. Verify Betfair API connectivity

### Rollback Procedures

**Immediate Rollback (< 15 minutes after deployment):**

```bash
# Rollback ALB to previous environment
cd terraform/environments/production
terraform apply -var="active_slot=blue"

# Verify rollback successful
curl https://api.clever-better.com/health
```

**Database Rollback:**

```bash
# List available snapshots
aws rds describe-db-snapshots --db-instance-identifier clever-better-prod

# Restore from snapshot
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier clever-better-prod-restored \
  --db-snapshot-identifier clever-better-prod-snapshot-YYYYMMDD

# Update application configuration to use restored database
# Requires terraform apply with updated database endpoint
```

### Database Migration Procedures

```bash
# Test migrations in dev environment first
cd /path/to/clever-better
make db-migrate-dev

# Review migration SQL
cat migrations/XXXX_migration_name.sql

# Apply to staging
make db-migrate-staging

# Apply to production (with caution)
make db-migrate-prod

# Verify migration success
make db-migration-status
```

**Migration Rollback:**
- All migrations should include DOWN/rollback scripts
- Test rollback in dev/staging before production deployment
- Maintain database backups before migration

### Configuration Updates

Configuration changes require redeployment:

```bash
# Update configuration in AWS Secrets Manager
aws secretsmanager update-secret \
  --secret-id clever-better/prod/config \
  --secret-string file://config/production.json

# Restart ECS services to pick up new config
aws ecs update-service \
  --cluster clever-better-prod \
  --service bot \
  --force-new-deployment
```

## Monitoring and Alerting

### Key Metrics to Monitor

**Trading Performance Metrics:**
- Current bankroll
- Daily P&L
- Win rate (target: >50%)
- ROI (target: >10%)
- Total exposure
- Bets placed per hour

**System Health Metrics:**
- ECS task health
- Database connections
- Memory usage (alert: >80%)
- CPU usage (alert: >75%)
- API response latency (p95 < 500ms)
- Error rate (alert: >1%)

**ML Service Metrics:**
- Prediction latency (p95 < 200ms)
- Model accuracy
- Cache hit ratio (target: >70%)
- Training job success rate

**Refer to:** `internal/metrics/metrics.go` for complete metrics definitions

### CloudWatch Dashboard Setup

Dashboards are automatically created via Terraform:

1. **Main Trading Dashboard**: Overview of trading performance
2. **ML Service Dashboard**: ML predictions and model performance
3. **Infrastructure Dashboard**: ECS, RDS, ALB metrics
4. **Security Dashboard**: GuardDuty findings, failed auth attempts

**Access Dashboards:**
```bash
# Open in browser
aws cloudwatch get-dashboard \
  --dashboard-name clever-better-prod-main

# Or via AWS Console
https://console.aws.amazon.com/cloudwatch/home?region=us-east-1#dashboards:
```

### Alert Thresholds and Escalation

**Critical Alerts (P0) - Immediate Response Required:**
- Emergency shutdown triggered
- Database connection failure
- Daily loss exceeds threshold (>$1000)
- Circuit breaker open for >30 minutes
- All ECS tasks unhealthy

**High Priority Alerts (P1) - Response within 15 minutes:**
- Error rate >5%
- API latency p95 >2 seconds
- ML service unavailable
- Memory usage >90%
- Betfair API auth failure

**Medium Priority Alerts (P2) - Response within 1 hour:**
- Error rate >2%
- Strategy performance degradation
- Cache hit ratio <50%
- Disk usage >80%

**Low Priority Alerts (P3) - Response within 4 hours:**
- Minor performance degradation
- Non-critical service warnings
- Configuration drift detected

**Escalation Path:**
1. On-call engineer (PagerDuty/Opsgenie)
2. Team lead (if unresolved in 30 minutes)
3. Engineering manager (if P0 unresolved in 1 hour)

### Log Aggregation and Analysis

**CloudWatch Log Groups:**
- `/ecs/clever-better-prod/bot`
- `/ecs/clever-better-prod/ml-service`
- `/ecs/clever-better-prod/data-ingestion`

**Common Log Queries:**

```bash
# Find errors in last hour
aws logs filter-log-events \
  --log-group-name /ecs/clever-better-prod/bot \
  --start-time $(date -u -d '1 hour ago' +%s)000 \
  --filter-pattern "ERROR"

# Find bet placements
aws logs filter-log-events \
  --log-group-name /ecs/clever-better-prod/bot \
  --filter-pattern "\"Bet placement recorded\""

# Check circuit breaker events
aws logs filter-log-events \
  --log-group-name /ecs/clever-better-prod/bot \
  --filter-pattern "\"circuit_breaker\""
```

### Performance Baselines

**Normal Operating Conditions:**
- API latency p95: 200-400ms
- Memory usage: 40-60%
- CPU usage: 20-40%
- Error rate: <0.5%
- Database connections: 10-20 active
- Bets per hour: 5-15
- ML prediction latency: 50-150ms

**Peak Load Conditions (race days):**
- API latency p95: 400-800ms
- Memory usage: 60-75%
- CPU usage: 40-65%
- Bets per hour: 20-50

## Incident Response

### Severity Classification

**P0 - Critical:**
- Complete system outage
- Data loss or corruption
- Security breach
- Financial loss >$5000/hour

**P1 - High:**
- Major feature unavailable
- Performance severely degraded
- Partial data loss
- Financial loss $1000-$5000/hour

**P2 - Medium:**
- Non-critical feature impaired
- Performance degraded
- Workaround available

**P3 - Low:**
- Minor inconvenience
- Cosmetic issues
- No immediate impact

**P4 - Informational:**
- Questions
- Feature requests
- Documentation updates

### On-Call Procedures

**When Alerted:**
1. Acknowledge alert within 5 minutes
2. Assess severity
3. Begin investigation
4. Update incident channel with status
5. Escalate if needed

**Incident Communication:**
- Create Slack incident channel: `#incident-YYYYMMDD-HHmm`
- Post initial status within 10 minutes
- Update every 30 minutes until resolved
- Notify stakeholders if P0/P1

**Tools:**
- PagerDuty for alerting
- Slack for communication
- Jira for incident tracking
- Zoom for war rooms (P0/P1)

### Post-Mortem Template

```markdown
# Incident Post-Mortem: [Brief Title]

**Date:** YYYY-MM-DD
**Duration:** X hours
**Severity:** P0/P1/P2/P3
**Incident Lead:** [Name]

## Summary
Brief description of the incident and impact.

## Timeline
- HH:MM - Incident detected
- HH:MM - Investigation began
- HH:MM - Root cause identified
- HH:MM - Fix deployed
- HH:MM - Incident resolved

## Root Cause
Detailed explanation of what caused the incident.

## Impact
- Users affected: X
- Revenue impact: $X
- Data affected: X records
- Services impacted: [list]

## Resolution
What was done to resolve the incident.

## Action Items
1. [Action] - Owner: [Name] - Due: YYYY-MM-DD
2. [Action] - Owner: [Name] - Due: YYYY-MM-DD

## Lessons Learned
- What went well
- What could be improved
- Process changes needed
```

### Common Incident Scenarios

**Scenario 1: Database Connection Exhaustion**
- **Symptoms:** High error rate, "connection pool exhausted" errors
- **Quick Fix:** Restart ECS tasks to reset connections
- **Long-term Fix:** Increase connection pool size, add connection timeouts

**Scenario 2: ML Service Unavailable**
- **Symptoms:** Prediction failures, timeouts
- **Quick Fix:** Restart ML service container
- **Long-term Fix:** Add circuit breaker, implement fallback strategy

**Scenario 3: Betfair API Rate Limiting**
- **Symptoms:** 429 errors, failed bet placements
- **Quick Fix:** Reduce request rate temporarily
- **Long-term Fix:** Implement backoff strategy, optimize API calls

**Scenario 4: High Memory Usage**
- **Symptoms:** OOM kills, slow performance
- **Quick Fix:** Restart affected tasks
- **Long-term Fix:** Identify memory leak, optimize caching

## Troubleshooting Procedures

Refer to [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed technical solutions.

### Trading Bot Not Placing Bets

**Check:**
1. Circuit breaker state: `aws logs filter-log-events --filter-pattern "circuit_breaker"`
2. Risk limits: Check current exposure vs. limits
3. Strategy status: Verify active strategies in database
4. Betfair connection: Check authentication status

### ML Service Prediction Failures

**Check:**
1. ML service health: `curl https://ml-service/health`
2. Model loaded: Check logs for model loading errors
3. Feature data: Verify input features are valid
4. Memory usage: Check if OOM issues

### Database Performance Degradation

**Check:**
1. Connection count: `SELECT count(*) FROM pg_stat_activity`
2. Long-running queries: Check pg_stat_activity
3. Index usage: Analyze query plans
4. Disk space: Verify storage not full

### Betfair API Connectivity Issues

**Check:**
1. API credentials: Verify not expired
2. SSL certificates: Check certificate validity
3. Network connectivity: Test from ECS task
4. Rate limits: Check request counts

### High Memory/CPU Usage

**Check:**
1. Container metrics in CloudWatch
2. Application logs for memory leaks
3. Recent deployments/changes
4. Database query performance

## Backup and Recovery

### Database Backup Schedule

**Automated Backups:**
- **Daily:** Full database backup at 02:00 UTC
- **Retention:** 30 days
- **Location:** AWS RDS automated backups

**Manual Snapshots:**
- Before production deployments
- Before major migrations
- Monthly for long-term retention

**Create Manual Snapshot:**
```bash
aws rds create-db-snapshot \
  --db-instance-identifier clever-better-prod \
  --db-snapshot-identifier clever-better-prod-manual-$(date +%Y%m%d)
```

### Model Backup Procedures

ML models stored in MLflow registry:

```bash
# Backup models to S3
aws s3 sync s3://clever-better-mlflow/models s3://clever-better-backups/models-$(date +%Y%m%d)

# List model versions
# Via MLflow UI or API
```

### Configuration Backup

```bash
# Backup Secrets Manager secrets
aws secretsmanager get-secret-value \
  --secret-id clever-better/prod/config \
  > backups/config-$(date +%Y%m%d).json

# Backup Terraform state
cd terraform/environments/production
terraform state pull > backups/terraform-state-$(date +%Y%m%d).json
```

### Disaster Recovery Plan

**RTO (Recovery Time Objective):** 4 hours  
**RPO (Recovery Point Objective):** 1 hour

**DR Procedure:**
1. Assess extent of disaster
2. Notify stakeholders
3. Restore database from latest snapshot (target: 30 minutes)
4. Deploy application to DR region (target: 1 hour)
5. Verify data integrity
6. Update DNS to point to DR region
7. Monitor and validate

**DR Regions:**
- Primary: us-east-1
- DR: us-west-2

## Maintenance Tasks

### Daily Tasks

- [ ] Review CloudWatch dashboards for anomalies
- [ ] Check trading performance (P&L, win rate)
- [ ] Review error logs
- [ ] Verify all ECS tasks healthy
- [ ] Check database performance metrics

### Weekly Tasks

- [ ] Review strategy performance
- [ ] Analyze ML model accuracy
- [ ] Check for security vulnerabilities
- [ ] Review and update on-call schedule
- [ ] Backup review and validation

### Monthly Tasks

- [ ] Database maintenance (vacuum, analyze)
- [ ] Cost optimization review
- [ ] Security audit
- [ ] Review and update documentation
- [ ] Disaster recovery drill

### Quarterly Tasks

- [ ] Dependency updates
- [ ] Performance testing
- [ ] Capacity planning review
- [ ] Architecture review
- [ ] Post-mortem review session

## Access and Permissions

### AWS Account Access

**Production Access:**
- Limited to senior engineers
- Requires MFA
- All actions logged in CloudTrail
- Review access quarterly

**Roles:**
- `DeploymentRole`: For CI/CD deployments
- `ReadOnlyRole`: For monitoring and troubleshooting
- `AdminRole`: For emergency access (break-glass)

### Database Credentials

**Rotation Schedule:**
- Application credentials: Every 90 days
- Admin credentials: Every 30 days

**Rotation Procedure:**
```bash
# Create new credentials in Secrets Manager
aws secretsmanager rotate-secret \
  --secret-id clever-better/prod/database

# Verify application still functional
# Update any manual connections
```

### Betfair API Key Management

- Keys stored in AWS Secrets Manager
- Rotation: Every 6 months
- Backup keys maintained
- Access logged

### Secrets Manager Usage

**Retrieve Secret:**
```bash
aws secretsmanager get-secret-value \
  --secret-id clever-better/prod/config \
  --query SecretString \
  --output text
```

**Update Secret:**
```bash
aws secretsmanager update-secret \
  --secret-id clever-better/prod/config \
  --secret-string file://new-config.json
```

## Runbook Maintenance

### Update Frequency

- Review quarterly
- Update after major incidents
- Update after architecture changes
- Update after process improvements

### Review Process

1. Quarterly review by on-call team
2. Validate procedures still accurate
3. Update based on feedback
4. Test critical procedures
5. Update version and date

### Version Control

- Runbook stored in git repository
- Changes reviewed via pull requests
- Version tagged with each update
- Previous versions accessible in git history

---

**Runbook Version:** 1.0.0  
**Last Updated:** 2026-02-03  
**Next Review:** 2026-05-03  
**Owner:** DevOps Team
