data "keycard_organization" "example" {}

# Look up a federated user by email, scoped to the identity provider that issued it
data "keycard_user" "alice" {
  zone_id = data.keycard_organization.example.zone_id
  email   = "alice@example.com"
  issuer  = "https://acme.okta.com"
}

# Look up a SCIM-provisioned user by the identity provider's subject
data "keycard_user" "bob" {
  zone_id = data.keycard_organization.example.zone_id
  subject = "00u1abcd2efGHIJK3l4m"
  issuer  = "https://acme.okta.com"
}

resource "keycard_group" "oncall" {
  zone_id = data.keycard_organization.example.zone_id
  name    = "On-call engineers"
}

# Add the resolved user to a group
resource "keycard_group_member" "alice_oncall" {
  zone_id  = data.keycard_organization.example.zone_id
  group_id = keycard_group.oncall.id
  user_id  = data.keycard_user.alice.id
}
