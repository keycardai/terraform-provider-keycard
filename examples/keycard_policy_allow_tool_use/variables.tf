variable "keycard_client_id" {
  description = "Keycard service account client ID"
  type        = string
}

variable "keycard_client_secret" {
  description = "Keycard service account client secret"
  type        = string
  sensitive   = true
}

variable "zone_id" {
  description = "ID of the zone whose active policy set will be updated"
  type        = string
}

variable "schema_version" {
  description = "Cedar schema version to validate the policy and policy set version against"
  type        = string
  default     = "2026-06-18"
}

variable "email_domain" {
  description = "Email domain the user must belong to"
  type        = string
  default     = "keycard.ai"
}
