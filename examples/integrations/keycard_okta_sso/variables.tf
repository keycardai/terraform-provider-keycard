# Keycard configuration
variable "keycard_client_id" {
  description = "Keycard service account client ID"
  type        = string
}

variable "keycard_client_secret" {
  description = "Keycard service account client secret"
  type        = string
  sensitive   = true
}

variable "keycard_endpoint" {
  description = "Keycard API endpoint"
  type        = string
  default     = "https://api.keycard.ai"
}

variable "organization_id" {
  description = "Keycard organization ID"
  type        = string
}

variable "keycard_redirect_uri" {
  description = "Keycard OAuth redirect URI for SSO"
  type        = string
  default     = "http://id.keycard.ai/oauth/2/redirect"
}

# Okta configuration
variable "okta_org_name" {
  description = "Okta organization name (the subdomain from your-org.okta.com)"
  type        = string
}

variable "okta_group_id" {
  description = "Okta group ID to assign to the SSO app (find via Okta Admin > Directory > Groups)"
  type        = string
}

variable "okta_base_url" {
  description = "Okta base URL (okta.com for production, oktapreview.com for preview)"
  type        = string
  default     = "okta.com"
}

variable "okta_api_token" {
  description = "Okta API token with admin privileges"
  type        = string
  sensitive   = true
}
