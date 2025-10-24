# Create a zone
resource "keycard_zone" "example" {
  name = "example-zone"
}

# Create a provider for credential issuance
resource "keycard_provider" "example" {
  name       = "Example Provider"
  identifier = "https://provider.example.com"
  zone_id    = keycard_zone.example.id
}

# Create an application that provides the resource
resource "keycard_application" "example" {
  name       = "Example Application"
  identifier = "https://app.example.com"
  zone_id    = keycard_zone.example.id
}

# Basic resource with required fields only
resource "keycard_resource" "basic" {
  name                   = "Basic Resource"
  identifier             = "https://api.example.com"
  zone_id                = keycard_zone.example.id
  credential_provider_id = keycard_provider.example.id
}

# Complete resource with all optional fields
resource "keycard_resource" "complete" {
  name                   = "Complete Resource"
  description            = "A resource with all optional fields configured"
  identifier             = "https://api.example.com/v2"
  zone_id                = keycard_zone.example.id
  credential_provider_id = keycard_provider.example.id
  application_id         = keycard_application.example.id

  metadata = {
    docs_url = "https://docs.example.com/api"
  }

  oauth2 = {
    scopes = ["read", "write", "admin"]
  }
}

# Resource with OAuth2 scopes
resource "keycard_resource" "with_scopes" {
  name                   = "API with Scopes"
  identifier             = "https://api.example.com/scoped"
  zone_id                = keycard_zone.example.id
  credential_provider_id = keycard_provider.example.id

  oauth2 = {
    scopes = ["my-api:read", "my-api:write"]
  }
}
