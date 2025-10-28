# Applications can be imported using the format: zones/{zone-id}/applications/{application-id}
import {
  to = keycard_application.basic
  id = "zones/zone-id-123/applications/application-id-456"
}

resource "keycard_application" "basic" {
  # Configuration will be populated after import
}
