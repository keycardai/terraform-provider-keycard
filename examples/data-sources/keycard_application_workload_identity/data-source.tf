# Look up an existing workload identity by zone_id and id
# Useful for referencing workload identities credentials created outside Terraform
data "keycard_application_workload_identity" "okta_mcp" {
  zone_id = keycard_zone.production.id
  id      = var.keycard_workload_identity_id
}

# Use the data source to verify configuration
output "mcp_service_account_subject" {
  value       = data.keycard_application_workload_identity.okta_mcp.subject
  description = "Kubernetes service account subject for Okta MCP server"
}
