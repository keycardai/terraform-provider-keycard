# Groups can be imported using the format: zones/{zone-id}/groups/{group-id}
import {
  to = keycard_group.example
  id = "zones/zone-id-123/groups/group-id-456"
}

resource "keycard_group" "example" {
  # Configuration will be populated after import
}
