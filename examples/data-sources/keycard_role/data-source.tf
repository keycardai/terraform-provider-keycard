data "keycard_organization" "example" {}

# Look up a role by ID
data "keycard_role" "admin" {
  zone_id = data.keycard_organization.example.zone_id
  id      = "7vq5j7fou4uz7fo93l8rjlzbm8"
}

# Look up a role by identifier; owner_type is required because an identifier is
# only unique per owner type within a zone
data "keycard_role" "viewer" {
  zone_id    = data.keycard_organization.example.zone_id
  identifier = "viewer"
  owner_type = "platform"
}

output "viewer_role_id" {
  value = data.keycard_role.viewer.id
}
