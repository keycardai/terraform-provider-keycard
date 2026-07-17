# List all Cedar policy schema versions available in a zone
data "keycard_policy_schemas" "all" {
  zone_id = keycard_zone.example.id
}

# List only the zone's default schema
data "keycard_policy_schemas" "default" {
  zone_id    = keycard_zone.example.id
  is_default = true
}

output "schema_versions" {
  value = data.keycard_policy_schemas.all.schemas[*].version
}
