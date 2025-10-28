# Create a zone, provider, and application first
resource "keycard_zone" "example" {
  name        = "my-zone"
  description = "Example zone for workload identities"
}

resource "keycard_application" "api" {
  name        = "My API"
  identifier  = "https://api.example.com"
  zone_id     = keycard_zone.example.id
  description = "Backend API application"
}

resource "keycard_provider" "eks" {
  name        = "AWS EKS Cluster"
  identifier  = "https://oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE"
  zone_id     = keycard_zone.example.id
  description = "EKS cluster OIDC provider"
}

# Workload identity with a specific subject constraint
resource "keycard_application_workload_identity" "eks_service_account" {
  zone_id        = keycard_zone.example.id
  application_id = keycard_application.api.id
  provider_id    = keycard_provider.eks.id
  subject        = "system:serviceaccount:default:my-service-account"
}

# Workload identity without subject - accepts any token from the provider
# Useful when you want to allow all workloads authenticated by the provider
resource "keycard_application_workload_identity" "eks_any" {
  zone_id        = keycard_zone.example.id
  application_id = keycard_application.api.id
  provider_id    = keycard_provider.eks.id
}
