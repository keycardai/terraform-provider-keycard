# Application URL credentials can be imported using the format: zones/{zone-id}/application-credentials/{credential-id}
import {
  to = keycard_application_url_credential.example
  id = "zones/zone-id-123/application-credentials/credential-id-abc"
}

resource "keycard_application_url_credential" "example" {
  # Configuration will be populated after import
}

