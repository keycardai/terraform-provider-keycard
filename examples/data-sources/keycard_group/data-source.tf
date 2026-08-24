data "keycard_organization" "example" {}

# Look up a group by ID
data "keycard_group" "engineering" {
  zone_id = data.keycard_organization.example.zone_id
  id      = keycard_group.engineering.id
}

# Look up a group by identifier, for groups created outside Terraform
data "keycard_group" "oncall" {
  zone_id    = data.keycard_organization.example.zone_id
  identifier = "oncall-engineers"
}

data "keycard_role" "viewer" {
  zone_id    = data.keycard_organization.example.zone_id
  identifier = "viewer"
  owner_type = "platform"
}

# Assign a role to the group found by identifier
resource "keycard_group_role_assignment" "oncall_viewer" {
  zone_id  = data.keycard_organization.example.zone_id
  group_id = data.keycard_group.oncall.id
  role_id  = data.keycard_role.viewer.id
}
