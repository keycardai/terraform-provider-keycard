data "keycard_policy_sets" "all" {
  zone_id = var.zone_id
}

locals {
  active_policy_set = one([
    for ps in data.keycard_policy_sets.all.policy_sets : ps
    if ps.active && ps.target_type == "zone"
  ])
}

data "keycard_policy_set_version" "current" {
  zone_id       = var.zone_id
  policy_set_id = local.active_policy_set.id
  id            = local.active_policy_set.active_version_id
}

data "keycard_policy_schema" "pinned" {
  zone_id = var.zone_id
  version = var.schema_version
}

resource "keycard_policy" "allow_tool_use" {
  zone_id     = var.zone_id
  name        = "allow-tool-use"
  description = "Allows tool use only for ${var.email_domain} users"
}

resource "keycard_policy_version" "allow_tool_use" {
  zone_id        = var.zone_id
  policy_id      = keycard_policy.allow_tool_use.id
  schema_version = data.keycard_policy_schema.pinned.version

  cedar = <<-CEDAR
    permit (
      principal is Keycard::User,
      action in [Keycard::Action::"WriteTools", Keycard::Action::"ControlTools"],
      resource
    )
    when { principal.email like "*@${var.email_domain}" };
  CEDAR

  lifecycle {
    create_before_destroy = true
  }
}

resource "keycard_policy_set_version" "allow_tool_use" {
  zone_id        = var.zone_id
  policy_set_id  = local.active_policy_set.id
  schema_version = data.keycard_policy_schema.pinned.version

  # Carry the active manifest forward and append the new policy. Filtering
  # out our own policy keeps the plan stable once this version is active.
  manifest = concat(
    [
      for entry in data.keycard_policy_set_version.current.manifest : {
        policy_id         = entry.policy_id
        policy_version_id = entry.policy_version_id
      } if entry.policy_id != keycard_policy.allow_tool_use.id
    ],
    [
      {
        policy_id         = keycard_policy.allow_tool_use.id
        policy_version_id = keycard_policy_version.allow_tool_use.id
      }
    ]
  )
}

resource "keycard_policy_set_activation" "zone" {
  zone_id               = var.zone_id
  policy_set_id         = local.active_policy_set.id
  policy_set_version_id = keycard_policy_set_version.allow_tool_use.id
}
