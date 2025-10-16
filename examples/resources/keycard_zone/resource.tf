# Basic zone configuration with just a name
resource "keycard_zone" "basic" {
  name = "my-zone"
}

# Zone with description
resource "keycard_zone" "with_description" {
  name        = "production-zone"
  description = "Production environment zone for customer-facing resources"
}
