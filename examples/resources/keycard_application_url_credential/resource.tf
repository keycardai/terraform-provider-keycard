# In order for an application to delegate authorization to a zone, it must be
# itself authorized to do so.
#
# Any application which publishes client ID metadata at a publicly available
# location may use keycard_application_url_credential for workload identity.
#
# See https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-00
# for more details on client ID metadata.
resource "keycard_application_url_credential" "keycard_mcp_gateway_auth" {
  zone_id        = keycard_zone.dev.id
  application_id = keycard_application.public_service.id
  url            = "https://public.example.com/oauth_client"
}

