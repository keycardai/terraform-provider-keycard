# Create a new Zone
resource "keycard_zone" "prod" {
  name = "Production"
}

# Fetch the directory for the Zone
data "keycard_zone_directory" "prod" {
  zone_id = keycard_zone.main.id
}

# Create a new MCP server resource that uses the Zone directory as
# the credential provider
resource "keycard_resource" "web_app" {
  zone_id                = keycard_zone.production.id
  name                   = "My MCP Server"
  description            = "MCP server protected by Keycard"
  identifier             = "https://mcp.my-product.com"
  credential_provider_id = data.keycard_zone_directory.provider_id

  metadata = {
    docs_url = "https://docs.my-product.com/mcp"
  }
}
