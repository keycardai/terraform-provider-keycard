output "policy_id" {
  description = "ID of the allow-tool-use policy"
  value       = keycard_policy.allow_tool_use.id
}

output "policy_version" {
  description = "Version number of the allow-tool-use policy version"
  value       = keycard_policy_version.allow_tool_use.version
}

output "policy_set_version" {
  description = "Version number of the new policy set version"
  value       = keycard_policy_set_version.allow_tool_use.version
}

output "activated_version_id" {
  description = "ID of the policy set version now active for the zone"
  value       = keycard_policy_set_activation.zone.policy_set_version_id
}
