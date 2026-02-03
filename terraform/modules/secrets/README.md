# Secrets Module

Creates Secrets Manager secrets for database, Betfair, and internal API keys.

## Rotation
- Optional rotation via Lambda ARN
- Schedule configured via rotation_days

## Populating Secrets
Use AWS CLI:
```
aws secretsmanager put-secret-value --secret-id <name> --secret-string '{"username":"...","password":"..."}'
```
