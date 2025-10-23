# Fetch the user identity provider configuration for a zone
data "keycard_zone_user_identity_config" "example" {
  zone_id = "etx6ju28wu5ibs3shgxqwwwpw0"
}

# Output the configured provider ID
output "identity_provider_id" {
  value = data.keycard_zone_user_identity_config.example.provider_id
}

# Use with a zone and provider resource
resource "keycard_zone" "my_zone" {
  name = "my-zone"
}

resource "keycard_provider" "okta" {
  zone_id    = keycard_zone.my_zone.id
  name       = "Okta"
  identifier = "https://dev-123456.okta.com"
}

resource "keycard_zone_user_identity_config" "my_config" {
  zone_id     = keycard_zone.my_zone.id
  provider_id = keycard_provider.okta.id
}

# Read back the configuration to verify
data "keycard_zone_user_identity_config" "lookup" {
  zone_id = keycard_zone_user_identity_config.my_config.zone_id
}
