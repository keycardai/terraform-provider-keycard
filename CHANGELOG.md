## 0.8.0

This release adds declarative group management: create zone-scoped groups, manage their membership, and grant roles to a group that every member inherits. Supporting data sources resolve the caller's organization, look up roles by identifier, and read existing groups.

FEATURES:

* **Groups**: New `keycard_group` resource and data source manage collections of users that can be assigned roles and referenced in policies.
* **Group Membership**: New `keycard_group_member` resource adds a user to a group, so the user inherits every role assigned to that group.
* **Group Role Assignments**: New `keycard_group_role_assignment` resource grants a role to a group.
* **Organization Lookup**: New `keycard_organization` data source returns the organization the authenticated identity belongs to, including its builtin `zone_id` — removing the need to hardcode a zone ID for organization-scoped configuration.
* **Role Lookup**: New `keycard_role` data source reads a role by `id`, or by `identifier` together with `owner_type` (`platform` or `customer`), so role assignments can reference roles by stable name rather than ID.

RESOURCES:

* `keycard_group` - New resource for managing groups.
* `keycard_group_member` - New resource adding a user to a group.
* `keycard_group_role_assignment` - New resource assigning a role to a group, with optional `scope_type`/`scope_id` for a scoped grant.

DATA SOURCES:

* `keycard_organization` - New data source returning the authenticated identity's organization: `id`, `name`, `label`, `sso_enabled`, and the builtin `zone_id`. Takes no arguments.
* `keycard_group` - New data source for looking up a group by `id` or `identifier`.
* `keycard_role` - New data source for reading a role by `id` or `identifier`.

## 0.7.0

This release adds end-to-end declarative Cedar policy management: author policies, compose them into sets, publish immutable versions, and promote an active set per zone — with data sources for lookup and listing at every level.

FEATURES:

* **Policy Schema Lookup**: New `keycard_policy_schema` data source resolves a zone's default Cedar policy schema version (or a specific version), laying the groundwork for declarative policy management.
* **Policy Containers**: New `keycard_policy` resource and data source manage Cedar policy containers within a zone. Destroy archives the policy, which frees its name for reuse.
* **Declarative Policy Authoring**: New `keycard_policy_version` resource publishes immutable Cedar policy content into a `keycard_policy`. Combined with the schema data source and `keycard_policy`, Cedar policies can now be authored end-to-end from Terraform, with a new immutable version created on every content change.
* **Policy Sets**: New `keycard_policy_set` resource and data source manage named compositions of policies, and the `keycard_policy_set_version` resource publishes immutable manifest snapshots pinning specific policy versions to a schema version.
* **Policy Set Activation**: New `keycard_policy_set_activation` resource binds a policy set version as the zone's active policy set, completing GitOps-style policy deployment: author Cedar, compose a set, and promote an immutable version to active — with rollback by re-pointing at a prior version.
* **Policy Version Lookup**: New `keycard_policy_version` and `keycard_policy_set_version` data sources read immutable versions by ID — including a set version's manifest — with archived versions surfaced via `archived_at`/`archived_by` rather than erroring.
* **Policy Listing**: New `keycard_policies`, `keycard_policy_sets`, and `keycard_policy_schemas` data sources enumerate a zone's policies, policy sets (with binding status), and Cedar schema versions, with optional server-side filters.

NOTES:

* `keycard_zone` - Create now blocks until the zone finishes provisioning across services (its PDP endpoints become reachable), waiting up to 2 minutes. This makes downstream resource creation and teardown reliable, but is a behavior change for configurations that create zones in isolation. Progress is logged at `INFO` (`TF_LOG=INFO`).

RESOURCES:

* `keycard_policy` - New resource for managing Cedar policy containers. Destroy archives the policy.
* `keycard_policy_version` - New resource for publishing immutable Cedar policy versions. Content and schema are `RequiresReplace` (versions are immutable); destroy archives the version.
* `keycard_policy_set` - New resource for managing policy set containers. Destroy archives the set.
* `keycard_policy_set_version` - New resource for publishing immutable policy set manifest snapshots. All attributes are `RequiresReplace`; destroy archives the version.
* `keycard_policy_set_activation` - New resource binding a policy set version as the zone's active policy set. Changing `policy_set_version_id` rolls forward (or back) in place. At most one activation per zone: the binding is keyed by the zone. Destroy does not deactivate (no deactivation API exists; a zone without an active binding fails closed) — it only removes the binding from state, and bound versions/sets cannot be archived while active.

DATA SOURCES:

* `keycard_policy_schema` - New data source for fetching a zone's Cedar policy schema by version or zone default.
* `keycard_policy` - New data source for looking up a policy by ID or name within a zone.
* `keycard_policy_set` - New data source for looking up a policy set by ID or name within a zone, including its binding state (active/shadow/latest version).
* `keycard_policy_version` - New data source for reading an immutable policy version by ID, including the server-normalized Cedar content. Archived versions read successfully with `archived_at`/`archived_by` populated.
* `keycard_policy_set_version` - New data source for reading an immutable policy set version by ID, including its manifest of `{policy_id, policy_version_id, sha}` entries. Archived versions read successfully with `archived_at`/`archived_by` populated.
* `keycard_policies` - New data source listing a zone's policies, with an optional server-side substring `name` filter.
* `keycard_policy_sets` - New data source listing a zone's policy sets with their binding status, with an optional server-side substring `name` filter.
* `keycard_policy_schemas` - New data source listing a zone's Cedar policy schema versions, with an optional `is_default` filter.
* `keycard_policy` (resource and data source) - Now exposes `created_at`, `updated_at`, `updated_by`, `latest_version` (numeric), and `latest_schema_version`.
* `keycard_policy_set` (resource and data source) - Now exposes `updated_by`.
* `keycard_policy_version` / `keycard_policy_set_version` (resources) - Now expose `archived_at`/`archived_by` (always null while managed; populated via the corresponding data sources).

DEPENDENCIES:

* Upgraded github.com/getkin/kin-openapi from 0.133.0 to 0.140.0
* Upgraded github.com/hashicorp/terraform-plugin-framework from 1.17.0 to 1.19.0
* Upgraded github.com/hashicorp/terraform-plugin-go from 0.29.0 to 0.31.0
* Upgraded github.com/hashicorp/terraform-plugin-testing from 1.14.0 to 1.16.0
* Upgraded github.com/oapi-codegen/runtime from 1.1.2 to 1.4.2 and the oapi-codegen tool from 2.5.0 to 2.7.2 (client regenerated)
* Upgraded golang.org/x/oauth2 from 0.35.0 to 0.36.0
* Upgraded google.golang.org/grpc to 1.79.3 and github.com/cloudflare/circl to 1.6.4 directly, removing the temporary `replace` directives
* Upgraded github.com/hashicorp/terraform-plugin-docs (tools) from 0.24.0 to 0.25.0, removing the goldmark and circl `replace` directives
* Updated all pinned GitHub Actions to current releases (checkout v7.0.0, setup-go v6.5.0, ghaction-import-gpg v7.0.0, goreleaser-action v7.2.3, setup-terraform v4.0.1, upload-artifact v7.0.1, golangci-lint-action v9.3.0)

## 0.6.1

This release adds support for resource prefix matching and includes some security patches in our software dependencies.

FEATURES:

* **Resource Prefix Matching**: `keycard_resource` supports a `prefix` attribute that causes the resource identifier to match any URL sharing it as a prefix at path/query/fragment boundaries. Enables protecting API roots and proxied hosts without declaring a resource per sub-path.

RESOURCES:

* `keycard_resource` - Added `prefix` attribute for URI-prefix matching. Defaults to `false`.

DATA SOURCES:

* `keycard_resource` - Added `prefix` computed attribute

SECURITY FIXES:

* Upgraded google.golang.org/grpc from 1.75.1 to 1.79.3 to patch an authorization bypass vulnerability from CVE-2026-33186.
* Upgraded github.com/cloudflare/circl go.mod 1.6.1 to 1.6.3 to fix an issue where the CombinedMult function would produce an incorrect value for specific inputs from CVE-2026-1229.
* Upgraded github.com/yuin/goldmark tools/go.mod 1.7.8 to 1.7.17 to patch a Cross-site Scripting vulnerability from CVE-2026-5160.

## 0.6.0

This release decouples the provider identifier from the OIDC issuer URL, enabling multiple providers to share the same issuer within a zone. It also adds IdP-initiated login support, application consent controls, and new data sources for SSO workflows.

FEATURES:

* **Provider Issuer Decoupling**: `keycard_provider` now supports an explicit `oauth2.issuer` field, separate from `identifier`. Existing configurations using `identifier` alone continue to work unchanged.
* **IdP-Initiated Login**: `keycard_sso_connection` exposes a computed `login_url` for configuring IdP-initiated login on your identity provider.
* **Application Consent**: `keycard_application` supports a `consent` attribute to control whether users are prompted with a consent screen (`required`) or consent is automatically granted (`implicit`) during authorization flows.

RESOURCES:

* `keycard_provider` - Added `oauth2.issuer` attribute for explicit OIDC issuer configuration. `identifier` is now optional when `oauth2.issuer` is provided and defaults to the issuer value. Schema upgraded to version 1 with automatic state migration.
* `keycard_sso_connection` - Added computed `login_url` attribute.
* `keycard_application` - Added `consent` attribute to control consent behavior during authorization flows. Defaults to `required`. ([#75](https://github.com/keycardai/terraform-provider-keycard/pull/75))

DATA SOURCES:

* `keycard_sso_login_url` - New data source to compute the IdP-initiated login URL from an issuer, with optional `target_link_uri`
* `keycard_provider` - Added `oauth2.issuer` computed attribute
* `keycard_application` - Added `consent` computed attribute ([#75](https://github.com/keycardai/terraform-provider-keycard/pull/75))

DOCUMENTATION:

* Added "Identifier and Issuer" section to provider resource documentation with three usage patterns
* Okta integration guides updated to use `oauth2.issuer`
* SSO guide updated to use `keycard_sso_login_url` data source for IdP-initiated login configuration

## 0.5.0

This release adds support for public application credentials, enabling OAuth2 public client flows for single-page apps, mobile apps, and other clients that cannot securely store a client secret.

RESOURCES:

* `keycard_application_public_credential` - New resource for managing public application credentials (client ID only, no secret) for OAuth2 public clients ([#57](https://github.com/keycardai/terraform-provider-keycard/pull/57))

DOCUMENTATION:

* Added documentation and examples for `keycard_application_public_credential`

## 0.4.0

This release adds organization-level Single Sign-On (SSO) support, enabling secure authentication through external identity providers, plus AWS PrivateLink integration examples for enhanced network security.

FEATURES:

* **Organization SSO Support**: Organizations can now configure Single Sign-On authentication with external identity providers like Okta, Azure AD, and Google Workspace

RESOURCES:

* `keycard_sso_connection` - New resource for managing SSO connections to identity providers ([#53](https://github.com/keycardai/terraform-provider-keycard/pull/53))

DOCUMENTATION:

* Added Organization SSO configuration guide with step-by-step Okta integration instructions
* New AWS PrivateLink integration example demonstrating secure VPC endpoint configuration and private DNS resolution for Keycard services ([#50](https://github.com/keycardai/terraform-provider-keycard/pull/50))
* Examples for configuring SSO with multiple providers: Okta, Azure AD, and Google Workspace

## 0.3.0

This release adds support for application traits, conditional dependencies, and URL credentials for workload identity.

FEATURES:

* **Application Traits**: Applications can now be assigned traits that ascribe behaviors and characteristics, enabling trait-specific workflows
* **Conditional Dependencies**: Application dependencies can now be filtered to activate only when accessing specific resources provided by the application

RESOURCES:

* `keycard_application_url_credential` - New resource for managing URL credentials for application workload identity using OAuth client ID metadata documents
* `keycard_application` - Added `traits` attribute to assign behaviors and characteristics to applications
* `keycard_application_dependency` - Added `when_accessing` attribute to express conditional dependencies

DATA SOURCES:

* `keycard_application` - Added new `traits` attribute

DOCUMENTATION:

* Examples demonstrating new features

## 0.2.0

This release adds support for customer managed encryption keys and includes bug fixes for optional string attributes.

FEATURES:

* **New Data Source**: `keycard_aws_kms_key_policy` - Generate AWS KMS key policies for Keycard zones with proper IAM permissions ([#39](https://github.com/keycardai/terraform-provider-keycard/pull/39))

ENHANCEMENTS:

* resource/zone: Added `encryption_key` attribute for AWS KMS key configuration ([#31](https://github.com/keycardai/terraform-provider-keycard/pull/31), [#34](https://github.com/keycardai/terraform-provider-keycard/pull/34))
* datasource/zone: Added `encryption_key` attribute ([#31](https://github.com/keycardai/terraform-provider-keycard/pull/31))

BUG FIXES:

* Validate length of optional string attributes across all resources to prevent "Provider produced inconsistent result" errors when empty strings are provided ([#38](https://github.com/keycardai/terraform-provider-keycard/pull/38))

## 0.1.1

Fixes for Okta integration guide documentation.

DOCUMENTATION:

* Revised Okta integration guide with complete variables declarations
* Removed broken links from Okta integration guide
* Added missing MCP server resource to Okta integration guide

## 0.1.0

Initial release of the Terraform Keycard Provider.

FEATURES:

* **New Provider**: Terraform provider for managing Keycard resources
* **OAuth2 Authentication**: Full support for OAuth2 client credentials flow with automatic token refresh
* **Zone Management**: Create and manage Keycard zones with OAuth2 configuration
* **Application Management**: Full lifecycle management of applications and their configurations
* **Identity Provider Integration**: Configure and manage identity providers and user identity mappings
* **Resource Protection**: Define and manage protected resources within your zones
* **Workload Identity Federation**: Support for workload identity with JWT and OIDC configurations
* **Access Management**: Configure application dependencies and access grants
* **Comprehensive Data Sources**: Read-only access to all managed resources for data lookups

RESOURCES:

* `keycard_zone` - Manage Keycard zones
* `keycard_provider` - Configure identity and credential providers
* `keycard_zone_user_identity_config` - Link user identities to zones
* `keycard_application` - Manage applications
* `keycard_application_client_secret` - Manage OAuth2 client credentials
* `keycard_application_workload_identity` - Configure workload identity federation
* `keycard_resource` - Define protected resources
* `keycard_application_dependency` - Configure application access grants

DATA SOURCES:

* `keycard_zone` - Look up zone information
* `keycard_provider` - Look up provider configurations
* `keycard_zone_user_identity_config` - Look up identity configurations
* `keycard_application` - Look up application details
* `keycard_application_workload_identity` - Look up workload identity configurations
* `keycard_resource` - Look up resource definitions

DOCUMENTATION:

* Complete provider and resource documentation
* Okta integration guide with step-by-step instructions
* Examples for all resources and common use cases
