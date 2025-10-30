# Workload identity for a backend service runnig in EKS with specific service account
resource "keycard_application_workload_identity" "backend_service" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.backend_service.id
  provider_id    = keycard_provider.eks.id
  subject        = "system:serviceaccount:production:backend-service-sa"
}

# Workload identity without subject constraint
# Accepts any authenticated token from the EKS cluster
resource "keycard_application_workload_identity" "internal_tools_any" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.backend_service.id
  provider_id    = keycard_provider.eks.id
}
