output "application_id" {
  description = "The ID of the created MCP server application"
  value       = keycard_application.mcp_server.id
}

output "application_name" {
  description = "The name of the MCP server application"
  value       = keycard_application.mcp_server.name
}

output "application_identifier" {
  description = "The identifier (URL) of the MCP server application"
  value       = keycard_application.mcp_server.identifier
}

output "client_id" {
  description = "OAuth2 client ID for the MCP server"
  value       = keycard_application_client_secret.mcp_server.client_id
  sensitive   = true
}

output "client_secret" {
  description = "OAuth2 client secret for the MCP server"
  value       = keycard_application_client_secret.mcp_server.client_secret
  sensitive   = true
}

output "resource_id" {
  description = "The ID of the MCP server resource"
  value       = keycard_resource.mcp_server.id
}

output "resource_name" {
  description = "The name of the MCP server resource"
  value       = keycard_resource.mcp_server.name
}

output "resource_identifier" {
  description = "The identifier of the MCP server resource"
  value       = keycard_resource.mcp_server.identifier
}

output "oauth2_credentials" {
  description = "OAuth2 credentials as a JSON object (for use with external systems)"
  value = jsonencode({
    client_id     = keycard_application_client_secret.mcp_server.client_id
    client_secret = keycard_application_client_secret.mcp_server.client_secret
  })
  sensitive = true
}

output "zone_oauth2_issuer_uri" {
  description = "The OAuth2 issuer URI for the zone"
  value       = data.keycard_zone.zone.oauth2.issuer_uri
}

output "env_file_path" {
  description = "Path to the generated environment file"
  value       = local_file.mcp_env.filename
}
