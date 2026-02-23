# Create a zone
resource "keycard_zone" "dev" {
  name = "Development"
}

# Register an OAuth2 identity provider
resource "keycard_provider" "okta" {
  zone_id       = keycard_zone.dev.id
  name          = "Okta"
  client_id     = var.okta_oauth_client_id
  client_secret = var.okta_oauth_client_secret

  oauth2 = {
    issuer = "https://dev-12345.okta.com"
  }
}

# Configure the zone to use Okta for user authentication
resource "keycard_zone_user_identity_config" "dev" {
  zone_id     = keycard_zone.dev.id
  provider_id = keycard_provider.okta.id
}
