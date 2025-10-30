# Look up an existing zone by ID
# Useful for referencing zones created outside Terraform
data "keycard_zone" "production" {
  id = "my-zone-id"
}

# Use zone OAuth2 settings when configuring external OAuth applications
output "zone_redirect_uri" {
  description = "Redirect URI to use when configuring external OAuth apps"
  value       = data.keycard_zone.production.oauth2.redirect_uri
}

output "zone_issuer_uri" {
  description = "OAuth2 issuer URI for the zone"
  value       = data.keycard_zone.production.oauth2.issuer_uri
}
