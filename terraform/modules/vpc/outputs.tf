output "vpc_id" {
  value = aws_vpc.main.id
}

output "vpc_cidr" {
  value = aws_vpc.main.cidr_block
}

output "public_subnet_ids" {
  value = [for s in aws_subnet.public : s.id]
}

output "private_app_subnet_ids" {
  value = [for s in aws_subnet.private_app : s.id]
}

output "private_data_subnet_ids" {
  value = [for s in aws_subnet.private_data : s.id]
}

output "nat_gateway_ids" {
  value = [for n in aws_nat_gateway.main : n.id]
}

output "internet_gateway_id" {
  value = aws_internet_gateway.main.id
}

output "availability_zones" {
  value = local.azs
}

output "flow_logs_log_group_name" {
  value = var.enable_flow_logs ? aws_cloudwatch_log_group.flow_logs[0].name : ""
}
