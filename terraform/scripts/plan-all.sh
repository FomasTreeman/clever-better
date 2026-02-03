#!/usr/bin/env bash
set -euo pipefail

ENVS=(dev staging production)

for ENV in "${ENVS[@]}"; do
  echo "Planning $ENV"
  pushd "terraform/environments/$ENV" >/dev/null
  terraform init
  terraform plan -out "plan-${ENV}.tfplan"
  popd >/dev/null
  echo "$ENV plan complete"
  echo ""
done
