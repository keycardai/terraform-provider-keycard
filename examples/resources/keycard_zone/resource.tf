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

# Zone with customer-managed encryption key (AWS KMS)
#
# IMPORTANT: When updating the encryption key, you must have access to both the
# old and new keys during the update operation. Do not revoke permissions on the
# existing key until after the plan has been applied successfully.
resource "keycard_zone" "encrypted" {
  name        = "Encrypted Zone"
  description = "Zone with data encrypted using AWS KMS"

  encryption_key {
    aws {
      arn = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
    }
  }
}

# Using zone OAuth2 redirect URI with external OAuth providers
# The zone provides a redirect_uri that can be used when configuring OAuth apps
resource "okta_app_oauth" "example" {
  label         = "My Application"
  type          = "web"
  redirect_uris = [keycard_zone.production.oauth2.redirect_uri]
}
