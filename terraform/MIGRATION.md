# Migration Guide

## Import Existing Resources
Use terraform import to bring existing AWS resources into state.

Example:
```
terraform import aws_vpc.main vpc-12345678
```

## Best Practices
- Import one resource at a time
- Validate state after each import
- Store state in S3 with versioning

## State Management
- Always enable DynamoDB locking
- Avoid manual state edits
