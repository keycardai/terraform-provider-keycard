# Fetch an existing provider by zone_id and id
data "keycard_provider" "example" {
  zone_id = "etx6ju28wu5ibs3shgxqwwwpw0"
  id      = "4rte3f0v5mkr3htgkp2glkrg00"
}

# Output the provider details
output "provider_name" {
  value = data.keycard_provider.example.name
}

output "provider_identifier" {
  value = data.keycard_provider.example.identifier
}

output "provider_description" {
  value = data.keycard_provider.example.description
}

output "provider_client_id" {
  value = data.keycard_provider.example.client_id
}

output "provider_oauth2_endpoints" {
  value = {
    authorization_endpoint = data.keycard_provider.example.oauth2.authorization_endpoint
    token_endpoint         = data.keycard_provider.example.oauth2.token_endpoint
  }
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

data "keycard_provider" "lookup" {
  zone_id = keycard_provider.okta.zone_id
  id      = keycard_provider.okta.id
}
