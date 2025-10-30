# Google Calendar API resource
# Requires Google OAuth provider and specifies calendar scope
resource "keycard_resource" "google_calendar" {
  name                   = "Google Calendar"
  identifier             = "https://www.googleapis.com/calendar/v3"
  zone_id                = keycard_zone.production.id
  credential_provider_id = keycard_provider.google.id

  oauth2 = {
    scopes = ["https://www.googleapis.com/auth/calendar"]
  }
}

# Google Drive API resource
# Demonstrates full drive access scope
resource "keycard_resource" "google_drive" {
  name                   = "Google Drive"
  identifier             = "https://www.googleapis.com/drive/v3"
  zone_id                = keycard_zone.production.id
  credential_provider_id = keycard_provider.google.id

  oauth2 = {
    scopes = ["https://www.googleapis.com/auth/drive"]
  }
}

# Okta protected resource
# Uses custom Okta authorization server with custom scopes
resource "keycard_resource" "okta_api" {
  name                   = "Okta Protected API"
  identifier             = "https://api.example.com"
  zone_id                = keycard_zone.production.id
  credential_provider_id = keycard_provider.okta_credentials.id

  oauth2 = {
    scopes = ["api:read", "api:write"]
  }
}

# Resource bound to a specific application
# Useful when a resource is exclusively accessed by one application
resource "keycard_resource" "application_specific" {
  name                   = "Application-Specific Resource"
  identifier             = "https://internal-api.example.com"
  zone_id                = keycard_zone.production.id
  credential_provider_id = keycard_provider.okta_credentials.id
  application_id         = keycard_application.backend_service.id

  oauth2 = {
    scopes = ["internal:access"]
  }
}
