# Fetch the zone's default Cedar policy schema
data "keycard_policy_schema" "default" {
  zone_id = keycard_zone.example.id
}

# Fetch a specific schema version
data "keycard_policy_schema" "pinned" {
  zone_id = keycard_zone.example.id
  version = "2026-02-24"
}

output "default_schema_version" {
  value = data.keycard_policy_schema.default.version
}
