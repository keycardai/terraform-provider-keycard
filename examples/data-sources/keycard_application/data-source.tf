# Look up an existing application by zone_id and id
# Useful when you need to reference applications created outside Terraform
data "keycard_application" "google_mcp" {
  zone_id = keycard_zone.production.id
  id      = var.google_mcp_server_id
}

# Use the data source to create client credentials for local development
resource "keycard_application_client_secret" "local_dev" {
  zone_id        = keycard_zone.production.id
  application_id = data.keycard_application.google_mcp.id
}
