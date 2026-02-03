# Audit Module

This module provides comprehensive audit logging and threat detection using AWS CloudTrail and GuardDuty.

## Features

### CloudTrail
- **Multi-region trail** for comprehensive API logging
- **Log file validation** to detect tampering
- **S3 storage** with lifecycle policies (Glacier, Deep Archive)
- **CloudWatch Logs integration** for real-time analysis
- **Event selectors** for S3 data events (optional)
- **Insights** for anomaly detection (optional)
- **Long-term retention** (7 years default for compliance)

### GuardDuty
- **Threat detection** for AWS accounts and workloads
- **S3 protection** to detect suspicious S3 activity
- **EKS protection** for Kubernetes audit logs (optional)
- **Malware protection** for EC2/EBS (optional)
- **Continuous monitoring** with 15-minute finding updates
- **SNS alerts** for high-severity findings
- **Findings export** to S3 (optional)

### Security Alerts
- **SNS topic** for centralized alerting
- **Email notifications** for security events
- **EventBridge integration** for GuardDuty findings
- **CloudWatch alarms** for suspicious activities:
  - Unauthorized API calls
  - Root account usage
  - IAM policy changes

## Architecture

```
┌─────────────────────────────────────────┐
│        AWS API Calls                    │
└──────────────┬──────────────────────────┘
               │
               ▼
      ┌────────────────┐
      │   CloudTrail   │
      └────┬───────┬───┘
           │       │
           ▼       ▼
    ┌──────────┐ ┌──────────────┐
    │ S3 Bucket│ │  CloudWatch  │
    │  (Logs)  │ │  Log Group   │
    └──────────┘ └──────┬───────┘
         │              │
         │              ▼
         │       ┌──────────────┐
         │       │ Metric Filter│
         │       │   & Alarms   │
         │       └──────┬───────┘
         │              │
         ▼              ▼
    ┌─────────────────────────┐
    │   GuardDuty Detector    │
    │  (Threat Intelligence)  │
    └──────────┬──────────────┘
               │
               ▼
        ┌─────────────┐
        │ EventBridge │
        │    Rule     │
        └──────┬──────┘
               │
               ▼
         ┌───────────┐
         │ SNS Topic │
         │  (Email)  │
         └───────────┘
```

## Usage

```hcl
module "audit" {
  source = "../../modules/audit"

  environment  = "production"
  project_name = "clever-better"
  
  # CloudTrail configuration
  enable_cloudtrail             = true
  cloudtrail_retention_days     = 90
  cloudtrail_s3_retention_days  = 2555  # 7 years
  enable_cloudtrail_insights    = true
  enable_s3_data_events         = false
  
  # GuardDuty configuration
  enable_guardduty                    = true
  guardduty_finding_frequency         = "FIFTEEN_MINUTES"
  enable_guardduty_s3_protection      = true
  enable_guardduty_eks_protection     = false
  enable_guardduty_malware_protection = false
  enable_guardduty_export             = true
  
  # Alerts
  alert_email = "security@clever-better.com"
  
  tags = {
    Project    = "clever-better"
    ManagedBy  = "terraform"
    Compliance = "SOC2"
  }
}
```

## CloudTrail Configuration

### Log Storage

CloudTrail logs are stored in two locations:

1. **S3 Bucket**: Long-term archival (7 years default)
   - Versioning enabled
   - Server-side encryption (AES-256)
   - Lifecycle policy: 90 days → Glacier → 365 days → Deep Archive → 2555 days → Delete
   - Public access blocked

2. **CloudWatch Logs**: Real-time analysis (90 days default)
   - Metric filters for suspicious activities
   - CloudWatch alarms for immediate alerting
   - Shorter retention for cost optimization

### Multi-Region Trail

- **Coverage**: All AWS regions
- **Global services**: Included (IAM, CloudFront, Route53)
- **Log file validation**: Enabled (SHA-256 hashing)
- **Prevents**: Log tampering and deletion

### Event Selectors

#### Management Events (Always Enabled)
- API calls for creating, modifying, deleting AWS resources
- IAM actions
- Console sign-ins

#### Data Events (Optional - Additional Cost)
```hcl
enable_s3_data_events = true  # S3 object-level operations
```

**Cost**: ~$0.10 per 100,000 events

### CloudTrail Insights (Optional)

Detects unusual API activity:
- Anomalous resource provisioning
- Burst of IAM actions
- Gaps in periodic maintenance activities

**Cost**: ~$0.35 per 100,000 write events analyzed

## GuardDuty Configuration

### Threat Detection

GuardDuty analyzes:
- **VPC Flow Logs**: Network traffic patterns
- **DNS Logs**: DNS query patterns
- **CloudTrail Logs**: API call patterns
- **S3 Data Events**: S3 object access patterns (if enabled)
- **EKS Audit Logs**: Kubernetes API calls (if enabled)

### Finding Severity Levels

| Severity | Range | Action |
|----------|-------|--------|
| Low | 0.1-3.9 | Informational, may be false positive |
| Medium | 4.0-6.9 | Investigate, potential security issue |
| High | 7.0-8.9 | **Alert sent**, investigate immediately |
| Critical | 9.0-10.0 | **Alert sent**, urgent response required |

### Common Findings

- **UnauthorizedAccess**: Unauthorized access attempts
- **Recon**: Reconnaissance activity (port scanning, brute force)
- **Backdoor**: Backdoor or C2 communication
- **CryptoCurrency**: Cryptocurrency mining
- **Trojan**: Trojan detected
- **Exfiltration**: Data exfiltration attempts

### Protection Features

#### S3 Protection (Enabled by Default)
- Detects suspicious API calls to S3
- Identifies data exfiltration attempts
- Monitors for credential misuse

#### EKS Protection (Optional)
- Analyzes Kubernetes audit logs
- Detects unauthorized pod deployments
- Identifies privilege escalation

#### Malware Protection (Optional - Additional Cost)
- Scans EBS volumes attached to EC2 instances
- Triggered by GuardDuty findings
- **Cost**: ~$1 per GB scanned

## Security Alerts

### SNS Topic

All security alerts are sent to an SNS topic:
- GuardDuty findings (severity >= 4)
- Unauthorized API call attempts
- Root account usage
- IAM policy changes

### Email Subscription

**Note**: After deployment, you must confirm the email subscription:

1. Check the email inbox for `alert_email`
2. Click the "Confirm subscription" link
3. Verify subscription is active:

```bash
aws sns list-subscriptions-by-topic \
  --topic-arn <topic_arn>
```

### Alarm Actions

CloudWatch alarms trigger SNS notifications for:

1. **Unauthorized API Calls**: > 5 attempts in 5 minutes
2. **Root Account Usage**: Any root account activity
3. **IAM Policy Changes**: Any IAM policy modification

## Compliance

### SOC 2 Type II
- ✅ Comprehensive audit logging
- ✅ Log integrity validation
- ✅ Long-term log retention (7 years)
- ✅ Real-time threat detection
- ✅ Security alerting

### PCI-DSS
- ✅ Requirement 10: Track and monitor all access to network resources
- ✅ Requirement 10.5: Secure audit trails
- ✅ Requirement 10.7: Retain audit trail history for at least 1 year

### HIPAA
- ✅ Access logging and monitoring
- ✅ Audit controls
- ✅ Integrity controls

### GDPR
- ✅ Accountability and governance
- ✅ Security monitoring
- ✅ Breach detection

## Cost Optimization

### Development/Staging

```hcl
enable_cloudtrail_insights    = false  # Save ~$35/month
enable_s3_data_events         = false  # Save on event costs
enable_guardduty_malware_protection = false  # Save on scan costs
cloudtrail_retention_days     = 30     # Shorter CloudWatch retention
```

**Estimated Cost**: ~$10-15/month

### Production

```hcl
enable_cloudtrail_insights    = true   # Enhanced anomaly detection
enable_s3_data_events         = false  # Enable if needed
enable_guardduty_malware_protection = false  # Enable if using EC2
cloudtrail_retention_days     = 90     # Compliance requirement
cloudtrail_s3_retention_days  = 2555   # 7-year retention
```

**Estimated Cost**: ~$40-50/month (without malware protection)

### Cost Breakdown

| Service | Cost |
|---------|------|
| CloudTrail trail | $2/month |
| CloudTrail S3 storage | $1-2/month |
| CloudWatch Logs | $2-5/month |
| GuardDuty | $5-10/month (based on volume) |
| CloudTrail Insights | $35/month (if enabled) |
| S3 Data Events | Variable (per 100k events) |
| Malware Protection | Variable (per GB scanned) |

## Monitoring and Analysis

### View CloudTrail Logs

```bash
# List recent events
aws cloudtrail lookup-events \
  --max-results 10

# Search for specific event
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=RunInstances

# Query CloudWatch Logs Insights
aws logs start-query \
  --log-group-name /aws/cloudtrail/clever-better-production \
  --start-time $(date -u -d '1 hour ago' +%s) \
  --end-time $(date -u +%s) \
  --query-string 'fields @timestamp, eventName, userIdentity.userName 
    | filter errorCode exists
    | sort @timestamp desc'
```

### View GuardDuty Findings

```bash
# List findings
aws guardduty list-findings \
  --detector-id <detector_id> \
  --finding-criteria '{"Criterion":{"severity":{"Gte":4}}}'

# Get finding details
aws guardduty get-findings \
  --detector-id <detector_id> \
  --finding-ids <finding_id>

# Archive finding
aws guardduty archive-findings \
  --detector-id <detector_id> \
  --finding-ids <finding_id>
```

### CloudWatch Insights Queries

#### Top API Calls
```
fields eventName, userIdentity.userName, sourceIPAddress
| stats count() by eventName
| sort count desc
| limit 20
```

#### Failed Authentication Attempts
```
fields @timestamp, eventName, errorCode, userIdentity.userName, sourceIPAddress
| filter errorCode = "AccessDenied" or errorCode = "UnauthorizedOperation"
| sort @timestamp desc
```

#### IAM Changes
```
fields @timestamp, eventName, userIdentity.userName, requestParameters
| filter eventName like /IAM/
| sort @timestamp desc
```

## Incident Response

### GuardDuty Finding Response

1. **Receive Alert**: Email notification from SNS
2. **Assess Severity**: Review finding details in GuardDuty console
3. **Investigate**: Check CloudTrail logs for related API calls
4. **Remediate**: 
   - Disable compromised credentials
   - Isolate affected resources
   - Apply security patches
5. **Archive Finding**: Once resolved

### CloudTrail Alert Response

1. **Receive Alarm**: Email from CloudWatch alarm
2. **Review Logs**: Check CloudWatch Logs for event details
3. **Validate**: Determine if activity is legitimate
4. **Respond**:
   - Legitimate: Document and dismiss
   - Suspicious: Investigate further, involve security team
   - Malicious: Disable access, rotate credentials, review damage

## Best Practices

1. **Enable in all regions**: Use multi-region trail
2. **Encrypt logs**: Use KMS for additional encryption (optional)
3. **Separate log account**: Store logs in separate AWS account (for large orgs)
4. **Regular reviews**: Periodically review CloudTrail and GuardDuty findings
5. **Test alerts**: Generate sample findings to verify alerting works
6. **Automate remediation**: Use Lambda to auto-remediate certain findings
7. **Integrate with SIEM**: Forward logs to Splunk, Sumo Logic, etc.
8. **Maintain runbooks**: Document response procedures for common findings

## Troubleshooting

### CloudTrail Not Logging

1. Check S3 bucket policy allows CloudTrail to write
2. Verify IAM role for CloudWatch Logs has correct permissions
3. Ensure trail is enabled (`IsLogging: true`)
4. Check for bucket lifecycle policies deleting logs too quickly

### GuardDuty No Findings

1. GuardDuty may take 24-48 hours for initial findings
2. No findings is good (but verify it's enabled)
3. Generate sample findings: GuardDuty → Settings → Generate sample findings

### Not Receiving Email Alerts

1. Check SNS subscription is confirmed
2. Verify email isn't marked as spam
3. Check EventBridge rule is enabled
4. Test SNS topic manually:

```bash
aws sns publish \
  --topic-arn <topic_arn> \
  --message "Test message"
```

## Inputs

| Name | Type | Default | Description |
|------|------|---------|-------------|
| environment | string | required | Environment name (dev, staging, production) |
| project_name | string | "clever-better" | Project name for resource naming |
| enable_cloudtrail | bool | true | Enable AWS CloudTrail |
| enable_guardduty | bool | true | Enable AWS GuardDuty |
| cloudtrail_retention_days | number | 90 | CloudWatch logs retention (days) |
| cloudtrail_s3_retention_days | number | 2555 | S3 logs retention (days, 7 years) |
| enable_cloudtrail_insights | bool | false | Enable CloudTrail Insights (additional cost) |
| enable_s3_data_events | bool | false | Enable S3 data events (additional cost) |
| guardduty_finding_frequency | string | "FIFTEEN_MINUTES" | Finding update frequency |
| enable_guardduty_s3_protection | bool | true | Enable GuardDuty S3 protection |
| enable_guardduty_eks_protection | bool | false | Enable GuardDuty EKS protection |
| enable_guardduty_malware_protection | bool | false | Enable malware scanning (additional cost) |
| alert_email | string | required | Email address for security alerts |
| enable_guardduty_export | bool | false | Export findings to S3 |
| tags | map(string) | {} | Additional resource tags |

## Outputs

| Name | Description |
|------|-------------|
| cloudtrail_id | CloudTrail ID |
| cloudtrail_arn | CloudTrail ARN |
| cloudtrail_bucket_name | S3 bucket for CloudTrail logs |
| cloudtrail_log_group_name | CloudWatch log group for CloudTrail |
| guardduty_detector_id | GuardDuty detector ID |
| security_alerts_topic_arn | SNS topic ARN for alerts |
| security_alerts_topic_name | SNS topic name for alerts |

## References

- [AWS CloudTrail](https://aws.amazon.com/cloudtrail/)
- [AWS GuardDuty](https://aws.amazon.com/guardduty/)
- [CloudTrail Best Practices](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/best-practices-security.html)
- [GuardDuty Findings](https://docs.aws.amazon.com/guardduty/latest/ug/guardduty_findings.html)
- [CIS AWS Foundations Benchmark](https://d1.awsstatic.com/whitepapers/compliance/AWS_CIS_Foundations_Benchmark.pdf)
