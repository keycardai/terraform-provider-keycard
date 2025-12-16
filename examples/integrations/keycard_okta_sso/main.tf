terraform {
  required_providers {
    keycard = {
      source  = "keycardai/keycard"
      version = "~> 0.3"
    }
    okta = {
      source  = "okta/okta"
      version = "~> 4.0"
    }
  }
}

provider "keycard" {
  client_id     = var.keycard_client_id
  client_secret = var.keycard_client_secret
  endpoint      = var.keycard_endpoint
}

provider "okta" {
  org_name  = var.okta_org_name
  base_url  = var.okta_base_url
  api_token = var.okta_api_token
}


# Create OIDC app in Okta for SSO
resource "okta_app_oauth" "keycard_sso" {
  label                      = "Keycard SSO"
  type                       = "web"
  grant_types                = ["authorization_code"]
  redirect_uris              = [var.keycard_redirect_uri]
  response_types             = ["code"]
  token_endpoint_auth_method = "client_secret_basic"
}

# Assign group to the app
resource "okta_app_group_assignments" "keycard_sso" {
  app_id = okta_app_oauth.keycard_sso.id
  group {
    id = var.okta_group_id
  }
}

# Configure Keycard to use Okta for SSO
# The identifier is the Okta org URL (issuer)
resource "keycard_sso_connection" "okta" {
  organization_id = var.organization_id
  identifier      = "https://${var.okta_org_name}.${var.okta_base_url}"
  client_id       = okta_app_oauth.keycard_sso.client_id
  client_secret   = okta_app_oauth.keycard_sso.client_secret
}
