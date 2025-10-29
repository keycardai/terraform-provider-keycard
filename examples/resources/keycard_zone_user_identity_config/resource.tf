# Example zone to configure
resource "keycard_zone" "dev" {
  name = "Development"
}

# Okta Oauth Provider for user logins
resource "keycard_provider" "okta" {
  zone_id       = keycard_zone.dev.id
  name          = "Okta"
  identifier    = "https://integrator-5548280.okta.com"
  client_id     = var.okta_oauth_client_id
  client_secret = var.okta_oauth_client_secret
}

# Configure the zone to use Okta as the user identity provider
# Users will authenticate through Okta when accessing resources in this zone
resource "keycard_zone_user_identity_config" "production" {
  zone_id     = keycard_zone.dev.id
  provider_id = keycard_provider.okta.id
}
