resource "keycard_provider" "okta" {
  zone_id       = keycard_zone.example_zone.id
  name          = "Okta"
  description   = "Okta provider for user authentication"
  identifier    = "https://dev-123456.okta.com"
  client_id     = "okta-client-id"
  client_secret = "okta-client-secret"
}
