# Fetch the organization the authenticated identity belongs to
data "keycard_organization" "this" {}

# Use the organization's builtin zone to create resources scoped to it
resource "keycard_policy_set" "org_default" {
  zone_id     = data.keycard_organization.this.zone_id
  name        = "org-default"
  target_type = "zone"
}
