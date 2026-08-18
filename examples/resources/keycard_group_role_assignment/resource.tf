data "keycard_organization" "example" {}

# Look the role up by identifier; owner_type disambiguates identifiers that
# exist under both owner types
data "keycard_role" "viewer" {
  zone_id    = data.keycard_organization.example.zone_id
  identifier = "viewer"
  owner_type = "platform"
}

data "keycard_role" "admin" {
  zone_id    = data.keycard_organization.example.zone_id
  identifier = "admin"
  owner_type = "platform"
}

# Grant every member of the group the viewer role across the organization
resource "keycard_group_role_assignment" "org_viewer" {
  zone_id  = data.keycard_organization.example.zone_id
  group_id = keycard_group.engineering.id
  role_id  = data.keycard_role.viewer.id
}

resource "keycard_zone" "custom_zone" {
  name = "Custom Zone"
}

# Scope a grant to a single zone instead of the whole organization
resource "keycard_group_role_assignment" "custom_zone_admin" {
  zone_id    = data.keycard_organization.example.zone_id
  group_id   = keycard_group.engineering.id
  role_id    = data.keycard_role.admin.id
  scope_type = "zone"
  scope_id   = keycard_zone.custom_zone.id
}
