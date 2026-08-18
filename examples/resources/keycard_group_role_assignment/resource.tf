# Look the role up by identifier; owner_type disambiguates identifiers that
# exist under both owner types
data "keycard_role" "zone_viewer" {
  zone_id    = keycard_zone.production.id
  identifier = "zone_viewer"
  owner_type = "platform"
}

# Assign the role; every member of the group inherits it
resource "keycard_group_role_assignment" "engineering_viewer" {
  zone_id  = keycard_zone.production.id
  group_id = keycard_group.engineering.id
  role_id  = data.keycard_role.zone_viewer.id
}

# Scope a grant to a single resource instead of the whole zone
resource "keycard_group_role_assignment" "oncall_zone_viewer" {
  zone_id    = keycard_zone.production.id
  group_id   = keycard_group.oncall.id
  role_id    = data.keycard_role.zone_viewer.id
  scope_type = "zone"
  scope_id   = keycard_zone.production.id
}
