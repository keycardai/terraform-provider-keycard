# Fetch an existing resource by zone_id and id
data "keycard_resource" "by_id" {
  zone_id = "etx6ju28wu5ibs3shgxqwwwpw0"
  id      = "res123456789"
}

# Fetch an existing resource by zone_id and identifier
data "keycard_resource" "by_identifier" {
  zone_id    = "etx6ju28wu5ibs3shgxqwwwpw0"
  identifier = "https://api.example.com"
}

# Output the resource details
output "resource_name" {
  value = data.keycard_resource.by_id.name
}

output "resource_identifier" {
  value = data.keycard_resource.by_id.identifier
}

output "resource_description" {
  value = data.keycard_resource.by_id.description
}

output "resource_metadata" {
  value = data.keycard_resource.by_id.metadata
}

output "resource_oauth2_scopes" {
  value = data.keycard_resource.by_id.oauth2.scopes
}

output "resource_credential_provider_id" {
  value = data.keycard_resource.by_id.credential_provider_id
}

output "resource_application_id" {
  value = data.keycard_resource.by_id.application_id
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
data "keycard_resource" "lookup_by_id" {
  zone_id = keycard_resource.api.zone_id
  id      = keycard_resource.api.id
}

# Lookup the resource by identifier
data "keycard_resource" "lookup_by_identifier" {
  zone_id    = keycard_resource.api.zone_id
  identifier = keycard_resource.api.identifier
}
