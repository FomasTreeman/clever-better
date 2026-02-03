output "web_acl_id" {
  value = aws_wafv2_web_acl.this.id
}

output "web_acl_arn" {
  value = aws_wafv2_web_acl.this.arn
}

output "ip_allowlist_set_id" {
  value = aws_wafv2_ip_set.allowlist.id
}

output "ip_blocklist_set_id" {
  value = aws_wafv2_ip_set.blocklist.id
}

output "log_bucket_name" {
  value = var.enable_logging ? aws_s3_bucket.waf_logs[0].bucket : ""
}
