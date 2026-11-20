# The application signs JWTs with its private key. Keycard verifies them
# using the public keys published at this JWKS URL.
resource "keycard_application_public_key_credential" "example" {
  zone_id        = keycard_zone.dev.id
  application_id = keycard_application.public_service.id
  jwks_uri       = "https://public.example.com/.well-known/jwks.json"
}

output "client_id" {
  value = keycard_application_public_key_credential.example.identifier
}
