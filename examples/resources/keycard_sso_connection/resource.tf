# Configure SSO with Okta for your organization
# This enables organization members to authenticate using Okta
resource "keycard_sso_connection" "okta" {
  organization_id = var.organization_id
  identifier      = "https://your-org.okta.com"
  client_id       = var.okta_client_id
  client_secret   = var.okta_client_secret
}

# Configure SSO with Azure AD
resource "keycard_sso_connection" "azure_ad" {
  organization_id = var.organization_id
  identifier      = "https://login.microsoftonline.com/${var.azure_tenant_id}/v2.0"
  client_id       = var.azure_client_id
  client_secret   = var.azure_client_secret
}

# Configure SSO with Google Workspace
resource "keycard_sso_connection" "google" {
  organization_id = var.organization_id
  identifier      = "https://accounts.google.com"
  client_id       = var.google_client_id
  client_secret   = var.google_client_secret
}
