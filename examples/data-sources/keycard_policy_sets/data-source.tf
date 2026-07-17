# List all policy sets in a zone with their binding status
data "keycard_policy_sets" "all" {
  zone_id = keycard_zone.example.id
}

# Find the currently bound (active) policy sets
output "active_policy_sets" {
  value = [for ps in data.keycard_policy_sets.all.policy_sets : ps.name if ps.active]
}
