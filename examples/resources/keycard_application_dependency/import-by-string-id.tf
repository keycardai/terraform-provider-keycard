# Application dependencies can be imported using the format: zones/{zone-id}/applications/{application-id}/dependencies/{resource-id}
import {
  to = keycard_application_dependency.example
  id = "zones/zone-id-123/applications/application-id-456/dependencies/resource-id-789"
}

resource "keycard_application_dependency" "example" {
  # Configuration will be populated after import
}
