# Generate OAuth2 client credentials for an MCP server
resource "keycard_application_client_secret" "google_mcp_server" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.google_mcp_server.id
}

# Output the credentials (marked as sensitive)
output "mcp_client_id" {
  description = "OAuth2 client ID for Google MCP server"
  value       = keycard_application_client_secret.google_mcp_server.client_id
  sensitive   = true
}

output "mcp_client_secret" {
  description = "OAuth2 client secret for Google MCP server"
  value       = keycard_application_client_secret.google_mcp_server.client_secret
  sensitive   = true
}

# Example: Store credentials in AWS Secrets Manager
resource "aws_secretsmanager_secret" "mcp_credentials" {
  name        = "google-mcp-oauth-credentials"
  description = "OAuth2 client credentials for Google MCP Server"
}

resource "aws_secretsmanager_secret_version" "mcp_credentials" {
  secret_id = aws_secretsmanager_secret.mcp_credentials.id
  secret_string = jsonencode({
    client_id     = keycard_application_client_secret.google_mcp_server.client_id
    client_secret = keycard_application_client_secret.google_mcp_server.client_secret
  })
}
