# Add a single user to a group
resource "keycard_group_member" "engineering_lead" {
  zone_id  = keycard_zone.production.id
  group_id = keycard_group.engineering.id
  user_id  = "user-id-123"
}

# Add several users to a group
resource "keycard_group_member" "oncall" {
  for_each = toset(["user-id-123", "user-id-456", "user-id-789"])

  zone_id  = keycard_zone.production.id
  group_id = keycard_group.oncall.id
  user_id  = each.value
}
