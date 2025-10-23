# Create a zone
resource "keycard_zone" "example" {
  name        = "Example Zone"
  description = "An example zone for demonstrating user identity provider configuration"
}

# Create a provider for user authentication
resource "keycard_provider" "okta" {
  zone_id       = keycard_zone.example.id
  name          = "Okta"
  description   = "Okta provider for user authentication"
  identifier    = "https://dev-123456.okta.com"
  client_id     = "okta-client-id"
  client_secret = "okta-client-secret"

  oauth2 = {
    authorization_endpoint = "https://dev-123456.okta.com/oauth2/v1/authorize"
    token_endpoint         = "https://dev-123456.okta.com/oauth2/v1/token"
  }
}

# Configure the zone to use this provider for user authentication
resource "keycard_zone_user_identity_config" "example" {
  zone_id     = keycard_zone.example.id
  provider_id = keycard_provider.okta.id
}
