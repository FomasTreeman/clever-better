# ECR Module

This module creates and manages Amazon Elastic Container Registry (ECR) repositories for the Clever Better project.

## Features

- Multiple repository creation with consistent naming
- Image scanning on push for security vulnerabilities
- Configurable lifecycle policies for automatic image cleanup
- Support for both AES256 and KMS encryption
- Repository policies for ECS task execution role access

## Usage

```hcl
module "ecr" {
  source = "../../modules/ecr"

  environment          = "dev"
  image_tag_mutability = "MUTABLE"

  # Optional: Grant ECS pull access
  ecs_execution_role_arn = module.iam.ecs_task_execution_role_arn

  tags = local.tags
}
```

## Docker Login

To authenticate Docker with ECR:

```bash
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  <account-id>.dkr.ecr.us-east-1.amazonaws.com
```

## Pushing Images

```bash
# Build image
docker build -t clever-better-bot:latest .

# Tag for ECR
docker tag clever-better-bot:latest \
  <account-id>.dkr.ecr.us-east-1.amazonaws.com/clever-better-dev-bot:latest

# Push to ECR
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/clever-better-dev-bot:latest
```

## Lifecycle Policy

The lifecycle policy automatically manages image retention:

1. **Tagged images**: Keeps the last N images (default: 10) with version prefixes
2. **Untagged images**: Expires after 7 days
3. **Latest tag**: Expires after 30 days to encourage proper versioning

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| environment | Environment name | string | - | yes |
| project_name | Project name | string | "clever-better" | no |
| repository_names | List of repos | list(string) | ["bot", "ml-service", "data-ingestion"] | no |
| image_tag_mutability | Tag mutability | string | "MUTABLE" | no |
| enable_scan_on_push | Enable scanning | bool | true | no |
| encryption_type | Encryption type | string | "AES256" | no |
| max_image_count | Max tagged images | number | 10 | no |
| ecs_execution_role_arn | ECS role ARN | string | "" | no |

## Outputs

| Name | Description |
|------|-------------|
| repository_urls | Map of repository URLs |
| repository_arns | Map of repository ARNs |
| repository_names | Map of full repository names |
| registry_id | ECR registry ID |
| registry_url | ECR registry URL |

## Environment-Specific Configuration

| Setting | Dev | Staging | Production |
|---------|-----|---------|------------|
| image_tag_mutability | MUTABLE | MUTABLE | IMMUTABLE |
| max_image_count | 10 | 10 | 20 |
| encryption_type | AES256 | AES256 | KMS |
