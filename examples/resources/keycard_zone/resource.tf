# Basic zone configuration
resource "keycard_zone" "development" {
  name = "Development"
}

# Production zone with description
resource "keycard_zone" "production" {
  name        = "Production"
  description = "Production environment for customer-facing applications"
}

# Zone with custom OAuth2 configuration
resource "keycard_zone" "custom" {
  name        = "Custom OAuth2"
  description = "Zone with customized OAuth2 settings"

  oauth2 {
    pkce_required = false # Disable PKCE requirement
    dcr_enabled   = true  # Enable Dynamic Client Registration
  }
}

# Using zone OAuth2 redirect URI with external OAuth providers
# The zone provides a redirect_uri that can be used when configuring OAuth apps
resource "okta_app_oauth" "example" {
  label         = "My Application"
  type          = "web"
  redirect_uris = [keycard_zone.production.oauth2.redirect_uri]
}
