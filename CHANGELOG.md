## Unreleased

FEATURES:

* **Policy Management**: New `keycard_policy` resource and data source manage Cedar authorization policies within a zone.
* **Policy Schema Lookup**: New `keycard_policy_schema` data source resolves a zone's default Cedar policy schema version (or a specific version), laying the groundwork for declarative policy management.

RESOURCES:

* `keycard_policy` - New resource for managing a Cedar policy in a zone. Supports import via `zones/{zone-id}/policies/{policy-id}`.

DATA SOURCES:

* `keycard_policy` - New data source for looking up a policy by `id` or `name` within a zone.
* `keycard_policy_schema` - New data source for fetching a zone's Cedar policy schema by version or zone default.

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
