variable "keycard_client_id" {
  description = "Keycard OAuth2 client ID"
  type        = string
}

variable "keycard_client_secret" {
  description = "Keycard OAuth2 client secret"
  type        = string
  sensitive   = true
}
