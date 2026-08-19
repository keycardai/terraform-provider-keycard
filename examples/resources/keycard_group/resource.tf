data "keycard_organization" "example" {}

# A group with an identifier derived from the name
resource "keycard_group" "engineering" {
  zone_id = data.keycard_organization.example.zone_id
  name    = "Engineering"
}

# A group with an explicit identifier, useful when policies or other systems
# reference the group by a stable name
resource "keycard_group" "oncall" {
  zone_id    = data.keycard_organization.example.zone_id
  name       = "On-Call Engineers"
  identifier = "oncall-engineers"
}
