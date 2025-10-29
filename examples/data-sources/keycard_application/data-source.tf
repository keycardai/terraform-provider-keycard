# Fetch an existing application by zone_id and id
data "keycard_application" "by_id" {
  zone_id = "etx6ju28wu5ibs3shgxqwwwpw0"
  id      = "app123456789"
}

# Fetch an existing application by zone_id and identifier
data "keycard_application" "by_identifier" {
  zone_id    = "etx6ju28wu5ibs3shgxqwwwpw0"
  identifier = "https://app.example.com"
}

# Output the application details
output "application_name" {
  value = data.keycard_application.by_id.name
}

output "application_identifier" {
  value = data.keycard_application.by_id.identifier
}

output "application_description" {
  value = data.keycard_application.by_id.description
}

output "application_metadata" {
  value = data.keycard_application.by_id.metadata
}

output "application_oauth2_redirect_uris" {
  value = data.keycard_application.by_id.oauth2.redirect_uris
}

# Use with an application resource
resource "keycard_zone" "production" {
  name = "production"
}

resource "keycard_application" "web_app" {
  zone_id     = keycard_zone.production.id
  name        = "Web Application"
  description = "Main production web application"
  identifier  = "https://app.example.com"

  metadata = {
    docs_url = "https://docs.example.com/web-app"
  }

  oauth2 = {
    redirect_uris = [
      "https://app.example.com/auth/callback",
      "https://app.example.com/oauth2/callback"
    ]
  }
}

# Lookup by ID
data "keycard_application" "lookup_by_id" {
  zone_id = keycard_application.web_app.zone_id
  id      = keycard_application.web_app.id
}

# Lookup by identifier
data "keycard_application" "lookup_by_identifier" {
  zone_id    = keycard_application.web_app.zone_id
  identifier = keycard_application.web_app.identifier
}
