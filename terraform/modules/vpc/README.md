# VPC Module

Creates a VPC with public, private application, and private data subnets across two AZs. Includes IGW, NAT gateways, and VPC Flow Logs.

## Inputs
- vpc_cidr
- environment
- enable_nat_gateway
- enable_flow_logs
- subnet CIDRs

## Outputs
- vpc_id
- subnet IDs
- nat_gateway_ids
- flow_logs_log_group_name

## Example
```hcl
module "vpc" {
  source = "../../modules/vpc"
  environment = var.environment
}
```
