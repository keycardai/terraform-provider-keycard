# Fetch an existing workload identity credential
data "keycard_application_workload_identity" "example" {
  zone_id = "zone-abc123"
  id      = "cred-xyz789"
}

# Use the data source outputs
output "workload_identity_subject" {
  value       = data.keycard_application_workload_identity.example.subject
  description = "The subject constraint for this workload identity"
}

output "workload_identity_provider" {
  value       = data.keycard_application_workload_identity.example.provider_id
  description = "The provider that validates tokens"
}
