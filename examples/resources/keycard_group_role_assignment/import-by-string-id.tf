# Group role assignments can be imported using the format: zones/{zone-id}/groups/{group-id}/roles/{assignment-id}
import {
  to = keycard_group_role_assignment.example
  id = "zones/zone-id-123/groups/group-id-456/roles/assignment-id-789"
}

resource "keycard_group_role_assignment" "example" {
  # Configuration will be populated after import
}
