output "sso_connection_id" {
  description = "The ID of the Keycard SSO connection"
  value       = keycard_sso_connection.okta.id
}

output "okta_app_client_id" {
  description = "The Okta OAuth app client ID"
  value       = okta_app_oauth.keycard_sso.client_id
  sensitive   = true
}

output "okta_issuer" {
  description = "The Okta issuer URL used for SSO"
  value       = "https://${var.okta_org_name}.${var.okta_base_url}"
}
