# Fetch an existing zone by ID
data "keycard_zone" "example" {
  id = "etx6ju28wu5ibs3shgxqwwwpw0"
}

# Output the zone details
output "zone_name" {
  value = data.keycard_zone.example.name
}

output "zone_description" {
  value = data.keycard_zone.example.description
}

output "zone_oauth2_settings" {
  value = {
    pkce_required = data.keycard_zone.example.oauth2.pkce_required
    dcr_enabled   = data.keycard_zone.example.oauth2.dcr_enabled
  }
}

# Use with a zone resource
resource "keycard_zone" "my_zone" {
  name = "my-zone"
}

data "keycard_zone" "lookup" {
  id = keycard_zone.my_zone.id
}
