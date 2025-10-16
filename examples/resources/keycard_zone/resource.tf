# Basic zone configuration with just a name
resource "keycard_zone" "basic" {
  name = "my-zone"
}

# Zone with description
resource "keycard_zone" "with_description" {
  name        = "production-zone"
  description = "Production environment zone for customer-facing resources"
}

# Zone with OAuth2 configuration
resource "keycard_zone" "with_oauth2" {
  name = "development-zone"

  oauth2 {
    pkce_required = true
    dcr_enabled   = true
  }
}

# Complete zone configuration with all options
resource "keycard_zone" "complete" {
  name        = "staging-zone"
  description = "Staging environment for testing before production"

  oauth2 {
    pkce_required = false
    dcr_enabled   = true
  }
}
