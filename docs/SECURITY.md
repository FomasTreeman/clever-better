# Security Documentation

This document describes the security architecture, policies, and best practices for Clever Better.

## Table of Contents

- [Overview](#overview)
- [Security Principles](#security-principles)
- [Authentication and Authorization](#authentication-and-authorization)
- [Secrets Management](#secrets-management)
- [Network Security](#network-security)
- [Data Security](#data-security)
- [Application Security](#application-security)
- [Monitoring and Incident Response](#monitoring-and-incident-response)
- [Compliance](#compliance)

## Overview

Security is critical for Clever Better as it handles:
- Financial trading operations
- Sensitive API credentials
- Personal Betfair account data

This document outlines the security controls implemented to protect these assets.

## Security Principles

1. **Defense in Depth**: Multiple layers of security controls
2. **Least Privilege**: Minimal permissions required for each component
3. **Zero Trust**: Verify explicitly, never assume trust
4. **Secure by Default**: Security controls enabled out of the box
5. **Fail Secure**: System fails in a secure state

## Authentication and Authorization

### Betfair API Authentication

Betfair uses certificate-based authentication:

```
Authentication Flow:
1. Present client certificate (SSL/TLS)
2. Send username/password with app key
3. Receive session token (SSOID)
4. Include SSOID in all subsequent requests
```

**Certificate Management:**
- Certificates stored in AWS Secrets Manager
- Rotated annually (or on compromise)
- Never committed to source control

**Session Management:**
```go
type BetfairSession struct {
    SSOID       string
    ExpiresAt   time.Time
    RefreshLock sync.Mutex
}

func (s *BetfairSession) Refresh() error {
    s.RefreshLock.Lock()
    defer s.RefreshLock.Unlock()

    if time.Until(s.ExpiresAt) > 30*time.Minute {
        return nil // Still valid
    }

    // Re-authenticate with certificate
    return s.authenticate()
}
```

### Internal Service Authentication

Services authenticate using:
- **gRPC**: Mutual TLS (mTLS) between Go and Python services
- **Database**: Username/password with SSL

### IAM Roles

Each component has a dedicated IAM role with minimal permissions:

**Bot Service Role:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": [
        "arn:aws:secretsmanager:*:*:secret:clever-better/*/betfair*",
        "arn:aws:secretsmanager:*:*:secret:clever-better/*/database*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:log-group:/ecs/clever-better-bot:*"
    }
  ]
}
```

## Secrets Management

### Secret Categories

| Secret | Storage | Rotation |
|--------|---------|----------|
| Betfair credentials | Secrets Manager | Annual |
| Betfair certificate | Secrets Manager | Annual |
| Database password | Secrets Manager | 90 days |
| API keys | Secrets Manager | On compromise |

### Secret Retrieval

```go
func GetSecret(ctx context.Context, secretName string) (string, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return "", err
    }

    client := secretsmanager.NewFromConfig(cfg)

    result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretName),
    })
    if err != nil {
        return "", err
    }

    return *result.SecretString, nil
}
```

### Local Development

For local development, use environment variables:

```bash
# Never commit .env files
export BETFAIR_APP_KEY="your-app-key"
export BETFAIR_USERNAME="your-username"
export BETFAIR_PASSWORD="your-password"
export DATABASE_URL="postgresql://localhost:5432/clever_better"
```

## Network Security

### VPC Architecture

```
Internet
    │
    ▼
[WAF] ─── Block malicious traffic
    │
    ▼
[ALB] ─── Public subnet, HTTPS only
    │
    ▼
[NAT] ─── Private subnet egress
    │
    ▼
[ECS] ─── Private application subnet
    │
    ▼
[RDS] ─── Isolated database subnet
```

### Security Groups

| Security Group | Inbound | Outbound |
|---------------|---------|----------|
| ALB | 443 from 0.0.0.0/0 | App SG:8000 |
| Application | 8000 from ALB SG | All (via NAT) |
| Database | 5432 from App SG | None |

### Network ACLs

Additional layer restricting:
- Inbound: Only HTTPS (443) to public subnets
- Outbound: Restricted to required ports (443, 5432)

### TLS Configuration

All connections use TLS 1.2 or higher:

```hcl
# ALB listener
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.main.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = aws_acm_certificate.main.arn
}
```

## Data Security

### Encryption at Rest

| Component | Encryption | Key Management |
|-----------|------------|----------------|
| RDS | AES-256 | AWS KMS |
| S3 | AES-256 | AWS KMS |
| EBS | AES-256 | AWS KMS |
| Secrets Manager | AES-256 | AWS KMS |

### Encryption in Transit

- All external connections: TLS 1.2+
- Internal service communication: mTLS
- Database connections: SSL required

### Data Classification

| Classification | Examples | Controls |
|---------------|----------|----------|
| Public | Documentation | None |
| Internal | Race data, odds | Access logging |
| Confidential | Trading history | Encryption, audit |
| Restricted | Credentials, PII | Encryption, strict access |

### Data Retention

| Data Type | Retention | Disposal |
|-----------|-----------|----------|
| Trade logs | 7 years | Secure deletion |
| Odds history | 2 years | Archival to Glacier |
| ML models | 1 year | Overwrite |
| Access logs | 1 year | Automated cleanup |

## Application Security

### Secure Coding Practices

**Input Validation:**
```go
func ValidateStake(stake float64) error {
    if stake <= 0 {
        return errors.New("stake must be positive")
    }
    if stake > config.MaxStakeLimit {
        return fmt.Errorf("stake exceeds limit of %.2f", config.MaxStakeLimit)
    }
    return nil
}
```

**SQL Injection Prevention:**
```go
// NEVER do this:
// query := fmt.Sprintf("SELECT * FROM races WHERE id = '%s'", raceID)

// Always use parameterized queries:
row := db.QueryRow("SELECT * FROM races WHERE id = $1", raceID)
```

**Error Handling:**
```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    result, err := processRequest(r)
    if err != nil {
        // Log full error internally
        log.Error("request failed", "error", err, "request_id", requestID)

        // Return generic error to client
        http.Error(w, "An error occurred", http.StatusInternalServerError)
        return
    }
}
```

### Dependency Security

```bash
# Go vulnerability scanning
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Python vulnerability scanning
pip install safety
safety check -r requirements.txt
```

### Container Security

```dockerfile
# Use minimal base image
FROM gcr.io/distroless/base-debian12

# Run as non-root user
USER nonroot:nonroot

# No shell access
ENTRYPOINT ["/app/bot"]
```

## Monitoring and Incident Response

### Security Monitoring

**CloudTrail**: All API calls logged
```json
{
  "eventSource": "secretsmanager.amazonaws.com",
  "eventName": "GetSecretValue",
  "userIdentity": { ... },
  "requestParameters": {
    "secretId": "clever-better/prod/betfair"
  }
}
```

**GuardDuty**: Threat detection enabled
- Unusual API activity
- Compromised EC2 instances
- Malicious IP connections

**CloudWatch Alarms:**
- Failed authentication attempts > 5/min
- Unusual trading volume
- API error rate spike

### Incident Response Plan

**Severity Levels:**

| Level | Description | Response Time |
|-------|-------------|---------------|
| P1 | Active compromise, data breach | Immediate |
| P2 | Potential compromise, anomaly detected | < 1 hour |
| P3 | Security policy violation | < 24 hours |
| P4 | Security improvement needed | < 1 week |

**Response Steps:**
1. **Contain**: Isolate affected systems
2. **Assess**: Determine scope and impact
3. **Remediate**: Fix vulnerability, rotate credentials
4. **Recover**: Restore normal operations
5. **Review**: Post-incident analysis

### Emergency Contacts

| Role | Contact | Escalation Time |
|------|---------|-----------------|
| On-call Engineer | PagerDuty | Immediate |
| Security Lead | [email] | 15 minutes |
| Management | [email] | 1 hour |

## Compliance

### Security Checklist

- [ ] All secrets in Secrets Manager
- [ ] No credentials in source code
- [ ] TLS enabled on all connections
- [ ] Database encryption enabled
- [ ] IAM roles follow least privilege
- [ ] Security groups properly configured
- [ ] Logging and monitoring enabled
- [ ] Vulnerability scanning automated
- [ ] Incident response plan documented

### Regular Security Tasks

| Task | Frequency | Owner |
|------|-----------|-------|
| Dependency updates | Weekly | Dev team |
| Vulnerability scan | Weekly | CI/CD |
| Access review | Monthly | Security |
| Penetration test | Annual | External |
| Credential rotation | 90 days | Automated |
| Security training | Annual | All staff |
