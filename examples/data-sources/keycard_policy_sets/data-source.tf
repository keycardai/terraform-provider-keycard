# List all policy sets in a zone with their binding status
data "keycard_policy_sets" "all" {
  zone_id = keycard_zone.example.id
}

# Find the zone's single active zone-scoped policy set. one() fails with a
# clear error if the zone has no active set.
locals {
  active_policy_set = one([
    for ps in data.keycard_policy_sets.all.policy_sets : ps
    if ps.active && ps.target_type == "zone"
  ])
}

output "active_policy_set_version_id" {
  value = local.active_policy_set.active_version_id
}
