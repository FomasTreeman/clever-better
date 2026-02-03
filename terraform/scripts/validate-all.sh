#!/usr/bin/env bash
set -euo pipefail

ENVS=(dev staging production)

echo "Validating modules"
for MODULE_DIR in terraform/modules/*; do
  if [[ -d "$MODULE_DIR" ]]; then
    echo "Validating module: $(basename "$MODULE_DIR")"
    pushd "$MODULE_DIR" >/dev/null
    terraform init -backend=false
    terraform fmt -check -recursive
    terraform validate
    if command -v tflint >/dev/null 2>&1; then
      tflint
    fi
    popd >/dev/null
  fi
done
echo "Modules validated"
echo ""

for ENV in "${ENVS[@]}"; do
  echo "Validating $ENV environment"
  pushd "terraform/environments/$ENV" >/dev/null
  
  # Validate with local backend first (checks syntax)
  terraform init -backend=false
  terraform fmt -check -recursive
  terraform validate
  if command -v tflint >/dev/null 2>&1; then
    tflint
  fi
  
  # Test remote backend initialization if AWS credentials are available
  if aws sts get-caller-identity >/dev/null 2>&1; then
    echo "Testing remote backend connectivity for $ENV..."
    # Initialize with backend (will fail gracefully if bucket doesn't exist yet)
    if terraform init -upgrade >/dev/null 2>&1; then
      echo "✓ Remote backend test passed for $ENV"
    else
      echo "⚠ Remote backend not fully initialized (may require setup-backend.sh to be run first)"
    fi
  fi
  
  popd >/dev/null
  echo "$ENV validated"
  echo ""
done

echo "All validations complete"
