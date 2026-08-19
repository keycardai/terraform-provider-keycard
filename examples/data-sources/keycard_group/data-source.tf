# Look up a group by ID
data "keycard_group" "engineering" {
  zone_id = keycard_zone.production.id
  id      = keycard_group.engineering.id
}

# Look up a group by identifier, for groups created outside Terraform
data "keycard_group" "oncall" {
  zone_id    = keycard_zone.production.id
  identifier = "oncall-engineers"
}

# Assign a role to the group found by identifier
resource "keycard_group_role_assignment" "oncall_viewer" {
  zone_id         = keycard_zone.production.id
  group_id        = data.keycard_group.oncall.id
  role_identifier = "zone_viewer"
  owner_type      = "platform"
}
