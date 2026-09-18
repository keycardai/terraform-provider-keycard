# Basic MCP Server Example
# This example shows the minimal configuration needed to set up an MCP server

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

# Create a zone for development
resource "keycard_zone" "dev" {
  name = "Development"
}

# Basic MCP server setup
module "mcp_server" {
  source = "../../"

  zone_id          = keycard_zone.dev.id
  mcp_server_url   = "https://my-mcp-server.example.com"
  application_name = "My MCP Server"
  env_file_name    = "mcp-server.env"
}

# Output the credentials
output "client_id" {
  description = "OAuth2 client ID for MCP server"
  value       = module.mcp_server.client_id
  sensitive   = true
}

output "client_secret" {
  description = "OAuth2 client secret for MCP server"
  value       = module.mcp_server.client_secret
  sensitive   = true
}
