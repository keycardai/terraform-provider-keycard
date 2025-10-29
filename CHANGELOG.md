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
