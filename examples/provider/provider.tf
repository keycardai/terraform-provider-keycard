terraform {
  required_providers {
    keycard = {
      source  = "keycardai/keycard"
      version = "~> 0.1"
    }
  }
}

provider "keycard" {
  client_id     = var.keycard_client_id
  client_secret = var.keycard_client_secret
}
