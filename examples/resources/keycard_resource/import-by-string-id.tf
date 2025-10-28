# Resources can be imported using the format: zones/{zone-id}/resources/{resource-id}
import {
  to = keycard_resource.example
  id = "zones/zone-id-123/resources/resource-id-789"
}

resource "keycard_resource" "example" {
  # Configuration will be populated after import
}
