#!/usr/bin/env bash
set -euo pipefail

read -rp "Environment (dev|staging|production): " ENV
read -rp "AWS region (e.g., us-east-1): " REGION

BUCKET="clever-better-terraform-state-${ENV}-${REGION}"
TABLE="clever-better-terraform-locks-${ENV}"

read -rp "This will delete backend resources. Type 'DELETE' to continue: " CONFIRM
if [[ "$CONFIRM" != "DELETE" ]]; then
  echo "Aborted."
  exit 1
fi

aws s3 rm "s3://$BUCKET" --recursive
aws s3api delete-bucket --bucket "$BUCKET" --region "$REGION"
aws dynamodb delete-table --table-name "$TABLE" --region "$REGION"

echo "Backend resources deleted."
