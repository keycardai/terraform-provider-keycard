# Providers can be imported using the format: zones/{zone-id}/providers/{provider-id}
import {
  to = keycard_provider.okta
  id = "zones/zone-id-123/providers/provider-id-xyz"
}

resource "keycard_provider" "okta" {
  # Configuration will be populated after import
}
