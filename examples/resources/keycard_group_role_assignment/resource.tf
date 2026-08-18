data "keycard_organization" "example" {}

# Look the role up by identifier; owner_type disambiguates identifiers that
# exist under both owner types
data "keycard_role" "viewer" {
  zone_id    = data.keycard_organization.example.zone_id
  identifier = "viewer"
  owner_type = "platform"
}

# Grant every member of the group the viewer role across the organization
resource "keycard_group_role_assignment" "org_viewer" {
  zone_id  = data.keycard_organization.example.zone_id
  group_id = keycard_group.engineering.id
  role_id  = data.keycard_role.viewer.id
}
