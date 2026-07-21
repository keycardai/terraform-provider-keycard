# List all policies in a zone
data "keycard_policies" "all" {
  zone_id = keycard_zone.example.id
}

# List policies whose name contains "admin" (case-insensitive substring match)
data "keycard_policies" "admin" {
  zone_id = keycard_zone.example.id
  name    = "admin"
}

# Exact-match filter (client-side): the name filter is a substring match, so
# "admin" also matches "admin-eu" and "administrator"
locals {
  admin_policy = one([
    for p in data.keycard_policies.admin.policies : p if p.name == "admin"
  ])
}

output "policy_names" {
  value = data.keycard_policies.all.policies[*].name
}
