# Example zone to create providers in
resource "keycard_zone" "dev" {
  name = "Development"
}

# EKS Workload Identity Provider (no credentials required)
# Uses the EKS cluster's OIDC provider URL
resource "keycard_provider" "eks" {
  zone_id    = keycard_zone.dev.id
  name       = "EKS Workload Identity"
  identifier = "https://oidc.eks.us-east-1.amazonaws.com/id/36DE0F31B34D3E28E0C538BDDCF86C98"
}

# Google OAuth Provider for accessing Google services
resource "keycard_provider" "google" {
  zone_id       = keycard_zone.dev.id
  name          = "Google"
  identifier    = "https://accounts.google.com"
  client_id     = var.google_oauth_client_id
  client_secret = var.google_oauth_client_secret
}

# Okta Oauth Provider for user logins
resource "keycard_provider" "okta" {
  zone_id       = keycard_zone.dev.id
  name          = "Okta"
  identifier    = "https://integrator-5548280.okta.com"
  client_id     = var.okta_oauth_client_id
  client_secret = var.okta_oauth_client_secret
}

# Configure the zone to use Okta as the user identity provider
# Users will authenticate through Okta when accessing resources in this zone
resource "keycard_zone_user_identity_config" "production" {
  zone_id     = keycard_zone.dev.id
  provider_id = keycard_provider.okta.id
}
