# Public credentials are suitable for public OAuth clients (e.g. single-page apps,
# mobile apps) that cannot keep a secret. A client ID (identifier) is auto-generated
# by Keycard and can be used as the OAuth 2.0 client_id in authorization requests.
resource "keycard_application_public_credential" "spa_client" {
  zone_id        = keycard_zone.dev.id
  application_id = keycard_application.spa.id
}

# The auto-generated client ID is available for use in other resources or outputs
output "spa_client_id" {
  description = "OAuth2 client ID for the single-page app"
  value       = keycard_application_public_credential.spa_client.client_id
}
