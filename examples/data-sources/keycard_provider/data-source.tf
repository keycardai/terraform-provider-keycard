# Fetch an existing provider by zone_id and id
data "keycard_provider" "by_id" {
  zone_id = "etx6ju28wu5ibs3shgxqwwwpw0"
  id      = "4rte3f0v5mkr3htgkp2glkrg00"
}

# Fetch an existing provider by zone_id and identifier
data "keycard_provider" "by_identifier" {
  zone_id    = "etx6ju28wu5ibs3shgxqwwwpw0"
  identifier = "https://dev-123456.okta.com"
}

# Fetch the default STS Provider for a Zone 
resource "keycard_zone" "example" {
  name = "Example Zone"
}

data "keycard_provider" "sts" {
  zone_id    = keycard_zone.example.id
  identifier = keycard_zone.example.oauth2.issuer_url
}

# Output the provider details
output "provider_name" {
  value = data.keycard_provider.by_id.name
}

output "provider_identifier" {
  value = data.keycard_provider.by_id.identifier
}

output "provider_description" {
  value = data.keycard_provider.by_id.description
}

output "provider_client_id" {
  value = data.keycard_provider.by_id.client_id
}

output "provider_oauth2_endpoints" {
  value = {
    authorization_endpoint = data.keycard_provider.by_id.oauth2.authorization_endpoint
    token_endpoint         = data.keycard_provider.by_id.oauth2.token_endpoint
  }
}

# Output STS Provider details
output "sts_provider_name" {
  value = data.keycard_provider.sts.name
}

output "sts_provider_identifier" {
  value       = data.keycard_provider.sts.identifier
  description = "The STS issuer URL"
}

# Use with a provider resource
resource "keycard_provider" "okta" {
  zone_id       = "etx6ju28wu5ibs3shgxqwwwpw0"
  name          = "Okta"
  description   = "Okta provider for user authentication"
  identifier    = "https://dev-123456.okta.com"
  client_id     = "okta-client-id"
  client_secret = "okta-client-secret"
}

# Lookup by ID
data "keycard_provider" "lookup_by_id" {
  zone_id = keycard_provider.okta.zone_id
  id      = keycard_provider.okta.id
}

# Lookup by identifier
data "keycard_provider" "lookup_by_identifier" {
  zone_id    = keycard_provider.okta.zone_id
  identifier = keycard_provider.okta.identifier
}
