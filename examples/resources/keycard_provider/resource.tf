# Create a zone
resource "keycard_zone" "dev" {
  name = "Development"
}

# Register an OAuth2 identity provider
resource "keycard_provider" "okta" {
  zone_id       = keycard_zone.dev.id
  name          = "Okta"
  client_id     = var.okta_oauth_client_id
  client_secret = var.okta_oauth_client_secret

  oauth2 = {
    issuer = "https://dev-12345.okta.com"
  }
}

# Entra correlates SCIM-provisioned users on "oid"; its pairwise "sub" differs
# from the SCIM externalId
resource "keycard_provider" "entra" {
  zone_id       = keycard_zone.dev.id
  name          = "Entra"
  client_id     = var.entra_oauth_client_id
  client_secret = var.entra_oauth_client_secret

  oauth2 = {
    issuer = "https://login.microsoftonline.com/${var.entra_tenant_id}/v2.0"
  }

  openid = {
    external_id_claim = "oid"
  }
}

# Configure the zone to use Okta for user authentication
resource "keycard_zone_user_identity_config" "dev" {
  zone_id     = keycard_zone.dev.id
  provider_id = keycard_provider.okta.id
}
