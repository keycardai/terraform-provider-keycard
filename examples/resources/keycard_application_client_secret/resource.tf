# Create a zone and application first
resource "keycard_zone" "example" {
  name        = "my-zone"
  description = "Example zone for applications"
}

resource "keycard_application" "example" {
  name        = "My Application"
  description = "Application that needs OAuth2 client credentials"
  identifier  = "https://myapp.example.com"
  zone_id     = keycard_zone.example.id
}

# Basic OAuth2 client credentials (client_id/client_secret)
resource "keycard_application_client_secret" "basic" {
  zone_id        = keycard_zone.example.id
  application_id = keycard_application.example.id
}

# Multiple credentials for the same application (e.g., for different environments)
resource "keycard_application_client_secret" "prod" {
  zone_id        = keycard_zone.example.id
  application_id = keycard_application.example.id
}

resource "keycard_application_client_secret" "staging" {
  zone_id        = keycard_zone.example.id
  application_id = keycard_application.example.id
}

# Output the credentials (marked as sensitive)
output "client_id" {
  description = "OAuth2 client ID"
  value       = keycard_application_client_secret.basic.client_id
  sensitive   = true
}

output "client_secret" {
  description = "OAuth2 client secret"
  value       = keycard_application_client_secret.basic.client_secret
  sensitive   = true
}

# Example: Store credentials in AWS Secrets Manager
resource "aws_secretsmanager_secret" "oauth_credentials" {
  name        = "myapp-oauth-credentials"
  description = "OAuth2 client credentials for MyApp"
}

resource "aws_secretsmanager_secret_version" "oauth_credentials" {
  secret_id = aws_secretsmanager_secret.oauth_credentials.id
  secret_string = jsonencode({
    client_id     = keycard_application_client_secret.basic.client_id
    client_secret = keycard_application_client_secret.basic.client_secret
    token_url     = "https://auth.keycard.ai/oauth2/token"
  })
}
