# Look up the user identity provider configuration for a zone
# Useful for verifying which provider is used for user authentication
data "keycard_zone_user_identity_config" "production" {
  zone_id = var.production_zone_id
}

# Output the configured identity provider
output "user_identity_provider" {
  description = "The provider used for user authentication in this zone"
  value       = data.keycard_zone_user_identity_config.production.provider_id
}
