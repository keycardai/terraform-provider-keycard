# A group with an identifier derived from the name
resource "keycard_group" "engineering" {
  zone_id = keycard_zone.production.id
  name    = "Engineering"
}

# A group with an explicit identifier, useful when policies or other systems
# reference the group by a stable name
resource "keycard_group" "oncall" {
  zone_id    = keycard_zone.production.id
  name       = "On-Call Engineers"
  identifier = "oncall-engineers"
}
