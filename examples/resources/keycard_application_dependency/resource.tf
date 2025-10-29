# Google MCP Server with multiple resource dependencies
# MCP servers typically need access to multiple APIs

# First dependency: Google Calendar
resource "keycard_application_dependency" "google_mcp_calendar" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.google_mcp_server.id
  resource_id    = keycard_resource.google_calendar.id
}

# Second dependency: Google Drive
resource "keycard_application_dependency" "google_mcp_drive" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.google_mcp_server.id
  resource_id    = keycard_resource.google_drive.id
}
