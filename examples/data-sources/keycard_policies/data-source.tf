# List all policies in a zone
data "keycard_policies" "all" {
  zone_id = keycard_zone.example.id
}

# List policies whose name contains "admin" (case-insensitive substring match)
data "keycard_policies" "admin" {
  zone_id = keycard_zone.example.id
  name    = "admin"
}

output "policy_names" {
  value = data.keycard_policies.all.policies[*].name
}
