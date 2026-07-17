# Read a policy version by ID, e.g. a policy's latest version
data "keycard_policy" "admin" {
  zone_id = keycard_zone.example.id
  name    = "admin-access"
}

data "keycard_policy_version" "latest" {
  zone_id   = keycard_zone.example.id
  policy_id = data.keycard_policy.admin.id
  id        = data.keycard_policy.admin.latest_version_id
}

output "cedar_content" {
  value = data.keycard_policy_version.latest.cedar
}
