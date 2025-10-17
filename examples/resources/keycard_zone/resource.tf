# Basic zone configuration with just a name
resource "keycard_zone" "basic" {
  name = "my-zone"
}

# Zone with description
resource "keycard_zone" "with_description" {
  name        = "production-zone"
  description = "Production environment zone for customer-facing resources"
}

# Zone with custom OAuth2 configuration
resource "keycard_zone" "with_oauth2" {
  name        = "custom-oauth2-zone"
  description = "Zone with customized OAuth2 settings"

  oauth2 {
    pkce_required = false # Disable PKCE requirement
    dcr_enabled   = true  # Enable Dynamic Client Registration
  }
}
