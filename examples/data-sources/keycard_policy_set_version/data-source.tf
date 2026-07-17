# Inspect the manifest of a policy set's active version
data "keycard_policy_set" "main" {
  zone_id = keycard_zone.example.id
  name    = "main"
}

data "keycard_policy_set_version" "active" {
  zone_id       = keycard_zone.example.id
  policy_set_id = data.keycard_policy_set.main.id
  id            = data.keycard_policy_set.main.active_version_id
}

output "active_manifest" {
  value = data.keycard_policy_set_version.active.manifest
}
