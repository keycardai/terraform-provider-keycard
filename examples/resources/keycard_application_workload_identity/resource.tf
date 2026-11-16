# Workload identity for a backend service runnig in EKS with specific service account
resource "keycard_application_workload_identity" "backend_service" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.backend_service.id
  provider_id    = keycard_provider.eks.id
  subject        = "system:serviceaccount:production:backend-service-sa"
}

# Workload identity for GitHub Actions, pinned to one organization
resource "keycard_application_workload_identity" "deploy" {
  zone_id                  = keycard_zone.production.id
  application_id           = keycard_application.backend_service.id
  provider_id              = keycard_provider.github.id
  subject                  = "repo:acme-corp/api-server:*"
  allowed_audience         = "https://github.com/acme-corp"
  authentication_condition = "claims.repository_owner == \"acme-corp\""
}

# The workload sends this as client_id alongside its token
output "deploy_client_id" {
  value = keycard_application_workload_identity.deploy.client_id
}
