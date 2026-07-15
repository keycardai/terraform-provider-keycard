# The policy set and an immutable version of it to activate.
resource "keycard_policy_set" "app" {
  zone_id = keycard_zone.production.id
  name    = "app-access"
}

resource "keycard_policy" "admin_access" {
  zone_id = keycard_zone.production.id
  name    = "admin-access"
}

data "keycard_policy_schema" "default" {
  zone_id = keycard_zone.production.id
}

resource "keycard_policy_version" "admin_access" {
  zone_id        = keycard_zone.production.id
  policy_id      = keycard_policy.admin_access.id
  schema_version = data.keycard_policy_schema.default.version

  cedar = <<-CEDAR
    permit (
      principal,
      action == Action::"read",
      resource
    );
  CEDAR

  lifecycle {
    create_before_destroy = true
  }
}

resource "keycard_policy_set_version" "app" {
  zone_id        = keycard_zone.production.id
  policy_set_id  = keycard_policy_set.app.id
  schema_version = data.keycard_policy_schema.default.version

  manifest = [
    {
      policy_id         = keycard_policy.admin_access.id
      policy_version_id = keycard_policy_version.admin_access.id
    }
  ]

  lifecycle {
    create_before_destroy = true
  }
}

# Binds the version as the zone's active policy set. Declare at most one
# activation per zone: the active binding is keyed by the zone, so activating a
# version of a different policy set takes over the slot. Pointing
# policy_set_version_id at another version (newer or older) re-activates in
# place. Destroying this resource does NOT deactivate anything; it only removes
# the binding from Terraform state.
resource "keycard_policy_set_activation" "app" {
  zone_id               = keycard_zone.production.id
  policy_set_id         = keycard_policy_set.app.id
  policy_set_version_id = keycard_policy_set_version.app.id
}
