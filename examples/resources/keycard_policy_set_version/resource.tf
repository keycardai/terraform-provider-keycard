# The policy set this version composes.
resource "keycard_policy_set" "app" {
  zone_id = keycard_zone.production.id
  name    = "app-access"
}

# A policy and an immutable version to reference from the manifest.
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

# An immutable manifest snapshot of the policy set. Changing any manifest entry
# or the schema publishes a new version. `create_before_destroy` publishes the
# replacement before the old version is archived. This resource does not
# activate the version; use keycard_policy_set_activation for that.
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
