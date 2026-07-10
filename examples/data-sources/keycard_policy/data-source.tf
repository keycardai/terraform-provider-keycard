# Look up a policy by ID
data "keycard_policy" "by_id" {
  zone_id = keycard_zone.example.id
  id      = "policy-id-789"
}

# Look up a policy by name
data "keycard_policy" "by_name" {
  zone_id = keycard_zone.example.id
  name    = "admin-access"
}

output "policy_owner_type" {
  value = data.keycard_policy.by_name.owner_type
}
