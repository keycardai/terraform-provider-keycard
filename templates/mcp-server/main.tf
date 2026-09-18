# MCP Server Terraform Module
# This module creates the necessary Keycard resources for an MCP server:
# - Application (represents the MCP server)
# - Resource (represents the MCP server as an API that can be accessed)
# - Application Client Secret (OAuth2 credentials for the MCP server)


data "keycard_zone" "zone" {
  id = var.zone_id
}

data "keycard_provider" "default" {
  zone_id    = var.zone_id
  identifier = data.keycard_zone.zone.oauth2.issuer_uri
}

resource "keycard_application" "mcp_server" {
  name        = var.application_name
  identifier  = var.mcp_server_url
  zone_id     = var.zone_id
  description = var.application_description
}

resource "keycard_resource" "mcp_server" {
  name                   = var.resource_name
  identifier             = var.mcp_server_url
  zone_id                = var.zone_id
  credential_provider_id = data.keycard_provider.default.id
  application_id         = keycard_application.mcp_server.id

}

resource "keycard_application_client_secret" "mcp_server" {
  zone_id        = var.zone_id
  application_id = keycard_application.mcp_server.id
}

resource "local_file" "mcp_env" {
  filename = var.env_file_name
  content  = <<-EOT
KEYCARD_ZONE_URL=${data.keycard_zone.zone.oauth2.issuer_uri}
KEYCARD_CLIENT_ID=${keycard_application_client_secret.mcp_server.client_id}
KEYCARD_CLIENT_SECRET=${keycard_application_client_secret.mcp_server.client_secret}
MCP_SERVER_URL=${keycard_resource.mcp_server.identifier}
EOT

  file_permission = "0600"
}
