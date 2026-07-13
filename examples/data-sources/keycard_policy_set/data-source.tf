# Look up a policy set by ID
data "keycard_policy_set" "by_id" {
  zone_id = keycard_zone.example.id
  id      = "policy-set-id-789"
}

# Look up a policy set by name
data "keycard_policy_set" "by_name" {
  zone_id = keycard_zone.example.id
  name    = "zone-default"
}

output "policy_set_active_version" {
  value = data.keycard_policy_set.by_name.active_version
}
