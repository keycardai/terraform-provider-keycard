# Allow tool use via the active policy set

Demonstrates the policy resource lifecycle end to end: author a Cedar policy that
allows tool-use actions only for users from a given email domain, cut a new
version of the zone's existing active policy set containing that policy, and
activate it — all in one apply.

## What it does

1. Finds the zone's currently active policy set
   (`data.keycard_policy_sets` filtered on `active`), its active version's manifest
   (`data.keycard_policy_set_version`), and the zone's default Cedar schema
   version (`data.keycard_policy_schema`).
2. Creates a policy (`keycard_policy.allow_tool_use`) and an immutable version
   of it (`keycard_policy_version.allow_tool_use`) whose Cedar body permits the
   `WriteTools` and `ControlTools` actions only when the principal's email
   matches `*@<var.email_domain>`.
3. Creates a new version of the existing policy set
   (`keycard_policy_set_version.allow_tool_use`) whose manifest is the active
   manifest plus the allow-tool-use policy version.
4. Activates the new version for the zone
   (`keycard_policy_set_activation.zone`).

The manifest expression filters out the new policy before appending it, so
re-planning after activation (when the active version already contains it) is a
no-op rather than an endless chain of new versions.

## Prerequisites

- A zone with an existing, **customer-owned** active policy set (creating a new
  version of a platform-owned set will be rejected).
- Keycard OAuth client credentials with policy management access to the zone.

## Usage

```sh
terraform init
terraform apply \
  -var zone_id=zone-123 \
  -var keycard_client_id=... \
  -var keycard_client_secret=...
```

## Caveats

- **Cedar permits are additive.** If the carried-forward manifest already
  contains a broader `permit` (e.g. allow-all), adding this policy does not
  restrict anything. To make this rule the only path to tool use, drop the
  `concat` and make it the sole manifest entry — Cedar is default-deny, so
  everything else is then denied.
- **Destroying `keycard_policy_set_activation` does not unbind anything.** A
  zone's active binding is replaced, never emptied, so destroy only removes the
  resource from Terraform state; the version stays active in the zone.
- **Rollback**: set `policy_set_version_id` back to the previously active
  version ID and apply — activation updates in place, including to older
  versions.
- **Teardown**: an active binding blocks archiving the resources it references,
  so `terraform destroy` cannot archive the active policy set version or the
  policy versions it references — they are dropped from Terraform state with a
  warning and stay live in the zone. Activate a different version first if you
  want them archived.
