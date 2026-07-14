# The policy container the version is published under.
resource "keycard_policy" "admin_access" {
  zone_id     = keycard_zone.production.id
  name        = "admin-access"
  description = "Grants administrative access to protected resources"
}

# The zone's default Cedar schema, used to validate the policy content.
data "keycard_policy_schema" "default" {
  zone_id = keycard_zone.production.id
}

# An immutable Cedar policy version. Editing `cedar` or `schema_version`
# publishes a new version. `create_before_destroy` ensures the replacement is
# published before the previous version is archived.
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
