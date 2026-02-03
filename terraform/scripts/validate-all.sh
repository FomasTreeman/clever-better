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
  echo "Validating $ENV"
  pushd "terraform/environments/$ENV" >/dev/null
  terraform init -backend=false
  terraform fmt -check -recursive
  terraform validate
  if command -v tflint >/dev/null 2>&1; then
    tflint
  fi
  popd >/dev/null
  echo "$ENV validated"
  echo ""
done
