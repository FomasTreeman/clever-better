#!/usr/bin/env bash
# Create ACM certificate for domain
# Usage: ./setup-acm-certificate.sh <domain-name> <region>

set -euo pipefail

DOMAIN=${1:-}
REGION=${2:-us-east-1}

if [[ -z "$DOMAIN" ]]; then
  echo "Usage: $0 <domain-name> <region>"
  exit 1
fi

aws acm request-certificate \
  --domain-name "$DOMAIN" \
  --validation-method DNS \
  --region "$REGION"

echo "Certificate requested."
echo "Next steps:"
echo "1) Describe the certificate to get DNS validation records:"
echo "   aws acm describe-certificate --certificate-arn <certificate-arn> --region $REGION"
echo "2) Add the DNS validation records to your DNS provider."
echo "3) Wait for certificate status to become ISSUED."
