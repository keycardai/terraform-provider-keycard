# Look up an existing resource by zone_id and id
# Useful for referencing resources created outside Terraform
data "keycard_resource" "google_calendar" {
  zone_id = keycard_zone.production.id
  id      = keycard_resource.google_calendar.id
}

# Create an application dependency using the data source
resource "keycard_application_dependency" "calendar_dependency" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.mobile_app.id
  resource_id    = data.keycard_resource.google_calendar.id
}
