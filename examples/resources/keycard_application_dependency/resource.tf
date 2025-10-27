# Create a zone for organizing resources
resource "keycard_zone" "example" {
  name        = "my-zone"
  description = "Example zone for applications and resources"
}

# Create a credential provider for the resource
resource "keycard_provider" "example" {
  name        = "Example OAuth2 Provider"
  description = "OAuth2 provider for resource authentication"
  identifier  = "https://auth.example.com"
  zone_id     = keycard_zone.example.id
}

# Create an application that needs to access resources
resource "keycard_application" "example_app" {
  name        = "My Application"
  description = "Application that needs delegated access to protected resources"
  identifier  = "https://myapp.example.com"
  zone_id     = keycard_zone.example.id
}

# Create a resource that the application needs to access
resource "keycard_resource" "example_resource" {
  name                   = "Protected API"
  description            = "API requiring authentication and authorization"
  identifier             = "https://api.example.com"
  zone_id                = keycard_zone.example.id
  credential_provider_id = keycard_provider.example.id
}

# Create an application dependency to allow the application to access the resource
# This enables the application to generate delegated user grants for accessing the resource
resource "keycard_application_dependency" "example_dep" {
  zone_id        = keycard_zone.example.id
  application_id = keycard_application.example_app.id
  resource_id    = keycard_resource.example_resource.id
}

# Example with multiple application dependencies for a single application
resource "keycard_resource" "second_resource" {
  name                   = "Data Service"
  description            = "Service providing access to user data"
  identifier             = "https://data.example.com"
  zone_id                = keycard_zone.example.id
  credential_provider_id = keycard_provider.example.id
}

resource "keycard_application_dependency" "second_dep" {
  zone_id        = keycard_zone.example.id
  application_id = keycard_application.example_app.id
  resource_id    = keycard_resource.second_resource.id
}
