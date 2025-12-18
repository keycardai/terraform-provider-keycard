# SSO connections can be imported using the organization ID or label
import {
  to = keycard_sso_connection.okta
  id = "org-12345"
}

resource "keycard_sso_connection" "okta" {
  # Configuration will be populated after import
  # Note: client_secret must be provided manually after import
}
