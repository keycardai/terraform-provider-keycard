# Group memberships can be imported using the format: zones/{zone-id}/groups/{group-id}/members/{user-id}
import {
  to = keycard_group_member.example
  id = "zones/zone-id-123/groups/group-id-456/members/user-id-789"
}

resource "keycard_group_member" "example" {
  # Configuration will be populated after import
}
