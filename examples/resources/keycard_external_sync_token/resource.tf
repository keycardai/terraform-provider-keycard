# Enable directory sync (SCIM) on the zone before minting a token
resource "keycard_zone_user_identity_config" "okta" {
  zone_id               = keycard_zone.org.id
  provider_id           = keycard_provider.okta.id
  external_sync_enabled = true
}

# Bearer token the identity provider presents to Keycard's SCIM endpoint
resource "keycard_external_sync_token" "okta" {
  zone_id    = keycard_zone.org.id
  depends_on = [keycard_zone_user_identity_config.okta]
}

output "scim_token" {
  value     = keycard_external_sync_token.okta.token
  sensitive = true
}
