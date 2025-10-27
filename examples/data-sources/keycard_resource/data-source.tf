# Fetch an existing resource by zone_id and id
data "keycard_resource" "example" {
  zone_id = "etx6ju28wu5ibs3shgxqwwwpw0"
  id      = "res123456789"
}

# Output the resource details
output "resource_name" {
  value = data.keycard_resource.example.name
}

output "resource_identifier" {
  value = data.keycard_resource.example.identifier
}

output "resource_description" {
  value = data.keycard_resource.example.description
}

output "resource_metadata" {
  value = data.keycard_resource.example.metadata
}

output "resource_oauth2_scopes" {
  value = data.keycard_resource.example.oauth2.scopes
}

output "resource_credential_provider_id" {
  value = data.keycard_resource.example.credential_provider_id
}

output "resource_application_id" {
  value = data.keycard_resource.example.application_id
}

# Use with a resource resource
resource "keycard_zone" "production" {
  name = "production"
}

resource "keycard_provider" "oauth_provider" {
  zone_id    = keycard_zone.production.id
  name       = "OAuth2 Provider"
  identifier = "https://oauth.example.com"
}

resource "keycard_application" "web_app" {
  zone_id    = keycard_zone.production.id
  name       = "Web Application"
  identifier = "https://app.example.com"
}

resource "keycard_resource" "api" {
  zone_id                = keycard_zone.production.id
  name                   = "API Resource"
  description            = "Main production API"
  identifier             = "https://api.example.com"
  credential_provider_id = keycard_provider.oauth_provider.id
  application_id         = keycard_application.web_app.id

  metadata = {
    docs_url = "https://docs.example.com/api"
  }

  oauth2 = {
    scopes = ["api:read", "api:write"]
  }
}

# Lookup the resource by ID
data "keycard_resource" "lookup" {
  zone_id = keycard_resource.api.zone_id
  id      = keycard_resource.api.id
}
