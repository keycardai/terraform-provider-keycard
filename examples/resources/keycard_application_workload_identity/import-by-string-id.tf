# Application workload identities can be imported using the format: zones/{zone-id}/application-credentials/{credential-id}
import {
  to = keycard_application_workload_identity.example
  id = "zones/zone-id-123/application-credentials/credential-id-abc"
}

resource "keycard_application_workload_identity" "example" {
  # Configuration will be populated after import
}
