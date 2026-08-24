# Group role assignments can be imported using the format: zones/{zone-id}/groups/{group-id}/roles/{assignment-id}
# Note this is the role assignment ID, not the role ID, because it is the only value that uniquely identifies a scoped grant.
terraform import keycard_group_role_assignment.example zones/zone-id-123/groups/group-id-456/roles/assignment-id-789
