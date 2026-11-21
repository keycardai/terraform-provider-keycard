# Google APIs MCP Server Example
# This example shows how to set up an MCP server with access to Google APIs

terraform {
  required_providers {
    keycard = {
      source  = "keycardai/keycard"
      version = "~> 0.1"
    }
  }
}

# Configure the Keycard provider
provider "keycard" {
  client_id     = var.keycard_client_id
  client_secret = var.keycard_client_secret
}

# Create a zone for production
resource "keycard_zone" "production" {
  name = "Production"
}

# MCP server setup
module "google_mcp_server" {
  source = "../../"

  zone_id                = keycard_zone.production.id
  mcp_server_url         = "https://google-mcp.example.com"
  application_name       = "Google APIs MCP Server"
  application_description = "MCP server providing access to Google APIs"
  resource_name          = "Google MCP Server API"
  env_file_name          = "google-mcp-server.env"

  # Define OAuth2 scopes for the MCP server API
  oauth2_scopes = [
    "https://www.googleapis.com/auth/calendar",
    "https://www.googleapis.com/auth/drive",
    "https://www.googleapis.com/auth/gmail.readonly",
    "https://www.googleapis.com/auth/gmail.send"
  ]
}

# Output the credentials for use in MCP server configuration
output "mcp_client_id" {
  description = "OAuth2 client ID for Google MCP server"
  value       = module.google_mcp_server.client_id
  sensitive   = true
}

output "mcp_client_secret" {
  description = "OAuth2 client secret for Google MCP server"
  value       = module.google_mcp_server.client_secret
  sensitive   = true
}

output "application_id" {
  description = "Application ID for the Google MCP server"
  value       = module.google_mcp_server.application_id
}

output "resource_id" {
  description = "ID of the MCP server resource"
  value       = module.google_mcp_server.resource_id
}
