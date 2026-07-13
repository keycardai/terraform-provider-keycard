# A policy set container within a zone. Versions and their bindings are
# managed separately through policy set versions.
resource "keycard_policy_set" "zone_default" {
  zone_id     = keycard_zone.production.id
  name        = "zone-default"
  target_type = "zone"
}
