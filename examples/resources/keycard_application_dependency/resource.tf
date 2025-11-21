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

# A multi-tasking agent application which provides many resources
# Not all resources need all dependencies
#
# The application depends on a grant only for the calendar API when providing a
# personal assistant.
resource "keycard_application_dependency" "personal_assistant_calendar" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.multi_tasking_agent.id
  resource_id    = keycard_resource.google_calendar.id
  when_accessing = [
    keycard_resource.personal_assistant.id
  ]
}

# The application depends on a grant only for the drive API when providing an
# archivist.
resource "keycard_application_dependency" "personal_archivist_drive" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.multi_tasking_agent.id
  resource_id    = keycard_resource.google_drive.id
  when_accessing = [
    keycard_resource.personal_archivist.id
  ]
}
