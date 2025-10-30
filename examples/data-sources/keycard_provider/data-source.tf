# Look up an existing provider by zone_id and identifier
# Useful for referencing providers created outside Terraform or by other teams
data "keycard_provider" "google" {
  zone_id    = keycard_zone.production.id
  identifier = "https://accounts.google.com"
}

# Use the provider in resource configurations
resource "keycard_resource" "google_photos" {
  name                   = "Google Photos"
  identifier             = "https://www.googleapis.com/photos/v1"
  zone_id                = keycard_zone.production.id
  credential_provider_id = data.keycard_provider.google.id

  oauth2 = {
    scopes = ["https://www.googleapis.com/auth/photoslibrary"]
  }
}
