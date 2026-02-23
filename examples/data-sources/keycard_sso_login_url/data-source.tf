# Compute the IdP-initiated login URL for an Okta SSO connection
# Use this to configure the login_uri on your identity provider's OAuth app
data "keycard_sso_login_url" "okta" {
  issuer = "https://your-org.okta.com"
}

# Optionally specify a custom target URI after login
data "keycard_sso_login_url" "okta_custom_target" {
  issuer          = "https://your-org.okta.com"
  target_link_uri = "https://console.keycard.ai/dashboard"
}
