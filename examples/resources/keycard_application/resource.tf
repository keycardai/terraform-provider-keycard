# Create a zone for the application
resource "keycard_zone" "example" {
  name        = "my-zone"
  description = "Example zone for applications"
}

# Basic application with minimal configuration
resource "keycard_application" "basic" {
  name       = "My Application"
  identifier = "https://myapp.example.com"
  zone_id    = keycard_zone.example.id
}

# Application with description and metadata
resource "keycard_application" "with_metadata" {
  name        = "Production Application"
  description = "Production application accessing protected APIs"
  identifier  = "https://prod-app.example.com"
  zone_id     = keycard_zone.example.id

  metadata {
    docs_url = "https://docs.example.com/prod-app"
  }
}

# Application with OAuth2 redirect URIs
resource "keycard_application" "with_oauth2" {
  name        = "Web Application"
  description = "Web application with OAuth2 authorization code flow"
  identifier  = "https://webapp.example.com"
  zone_id     = keycard_zone.example.id

  oauth2 {
    redirect_uris = [
      "https://webapp.example.com/callback",
      "https://webapp.example.com/auth/callback"
    ]
  }
}

# Complete application example with all optional fields
resource "keycard_application" "complete" {
  name        = "Complete Application"
  description = "Application demonstrating all available configuration options"
  identifier  = "https://complete.example.com"
  zone_id     = keycard_zone.example.id

  metadata {
    docs_url = "https://docs.example.com/complete-app"
  }

  oauth2 {
    redirect_uris = [
      "https://complete.example.com/callback"
    ]
  }
}

# Output application details
output "basic_app_id" {
  description = "ID of the basic application"
  value       = keycard_application.basic.id
}

output "oauth2_app_id" {
  description = "ID of the OAuth2 application"
  value       = keycard_application.with_oauth2.id
}
