# SSO connections can be imported using the format: organizations/{organization_id}/sso-connection
import {
  to = keycard_sso_connection.okta
  id = "organizations/org-12345/sso-connection"
}

resource "keycard_sso_connection" "okta" {
  # Configuration will be populated after import
  # Note: client_secret must be provided manually after import
}
