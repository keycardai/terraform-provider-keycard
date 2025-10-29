# Okta Integration Guide

This guide demonstrates a complete integration between Keycard and Okta, showing how to:

1. Configure Okta as a user identity provider
2. Set up Okta for credential management
3. Create OAuth applications in Okta
4. Configure authorization policies
5. Protect resources with Okta-issued credentials
6. Grant application access to protected resources

## Prerequisites

- **Keycard Account**: You need a Keycard account with service account credentials
- **Okta Account**: You need an Okta account with admin access

## Quick Start

### 1. Set Up Variables

Create a `terraform.tfvars` file with your configuration:

```hcl
# Keycard credentials
keycard_client_id     = "your-keycard-client-id"
keycard_client_secret = "your-keycard-client-secret"

# Okta credentials
okta_org_name  = "dev-12345"  # Your Okta organization name
okta_api_token = "your-okta-api-token"

# Optional: Customize these if needed
okta_custom_scope             = "api:access"
application_identifier        = "https://api.example.com"
okta_protected_resource_url   = "https://api.example.com/protected"
```

### 2. Apply Terraform

```hcl
terraform {
  required_providers {
    keycard = {
      source  = "keycardai/keycard"
      version = "~> 0.1"
    }
    okta = {
      source  = "okta/okta"
      version = "~> 6.4.0"
    }
  }
}

provider "keycard" {
  client_id     = var.keycard_client_id
  client_secret = var.keycard_client_secret
}

provider "okta" {
  org_name  = var.okta_org_name
  api_token = var.okta_api_token
}

# Step 1: Create a Keycard Zone
resource "keycard_zone" "okta_demo" {
  name        = "Okta Demo"
  description = "Demo environment with Okta integration"
}

# Step 2: Set up Okta as the Identity Provider
# Fetch Okta organization metadata
data "okta_org_metadata" "default" {}

# Create OAuth app in Okta for user authentication
resource "okta_app_oauth" "keycard_idp" {
  label               = "Keycard User Authentication"
  type                = "web"
  redirect_uris       = [keycard_zone.okta_demo.oauth2.redirect_uri]
  implicit_assignment = true
}

# Register Okta as an identity provider in Keycard
resource "keycard_provider" "okta_idp" {
  zone_id       = keycard_zone.okta_demo.id
  name          = "Okta Identity Provider"
  identifier    = data.okta_org_metadata.default.domains.organization
  client_id     = okta_app_oauth.keycard_idp.client_id
  client_secret = okta_app_oauth.keycard_idp.client_secret
}

# Configure the zone to use Okta for user authentication
resource "keycard_zone_user_identity_config" "okta_demo" {
  zone_id     = keycard_zone.okta_demo.id
  provider_id = keycard_provider.okta_idp.id
}

# Step 3: Set up Okta for Credential Management
# Get the default Okta authorization server
data "okta_auth_server" "default" {
  name = "default"
}

# Create OAuth app in Okta for credential management
resource "okta_app_oauth" "keycard_credentials" {
  label               = "Keycard Credentials"
  type                = "web"
  redirect_uris       = [keycard_zone.okta_demo.oauth2.redirect_uri]
  implicit_assignment = true
}

# Create authorization policy for Keycard
resource "okta_auth_server_policy" "keycard" {
  auth_server_id   = data.okta_auth_server.default.id
  status           = "ACTIVE"
  name             = "Keycard Policy"
  description      = "Authorization policy for Keycard applications"
  priority         = 1
  client_whitelist = [okta_app_oauth.keycard_credentials.client_id]
}

# Create policy rule to allow required grant types
resource "okta_auth_server_policy_rule" "keycard" {
  auth_server_id       = data.okta_auth_server.default.id
  policy_id            = okta_auth_server_policy.keycard.id
  status               = "ACTIVE"
  name                 = "Allow Keycard Access"
  priority             = 1
  grant_type_whitelist = ["authorization_code", "client_credentials"]
  scope_whitelist      = [var.okta_custom_scope]
  group_whitelist      = ["EVERYONE"]
}

# Register Okta as a credential provider in Keycard
resource "keycard_provider" "okta_credentials" {
  zone_id       = keycard_zone.okta_demo.id
  name          = "Okta Credential Provider"
  identifier    = data.okta_auth_server.default.issuer
  client_id     = okta_app_oauth.keycard_credentials.client_id
  client_secret = okta_app_oauth.keycard_credentials.client_secret
}

# Step 4: Create a Protected Resource
resource "keycard_resource" "okta_api" {
  name                   = "Okta Protected API"
  identifier             = var.okta_protected_resource_url
  zone_id                = keycard_zone.okta_demo.id
  credential_provider_id = keycard_provider.okta_credentials.id

  oauth2 = {
    scopes = [var.okta_custom_scope]
  }
}

# Step 5: Create an Application
resource "keycard_application" "api_server" {
  name        = "API Server"
  identifier  = var.application_identifier
  zone_id     = keycard_zone.okta_demo.id
  description = "Backend API server accessing Okta-protected resources"
}

# Generate client credentials for the application
resource "keycard_application_client_secret" "api_server" {
  zone_id        = keycard_zone.okta_demo.id
  application_id = keycard_application.api_server.id
}

# Grant the application access to the protected resource
resource "keycard_application_dependency" "api_server_okta" {
  zone_id        = keycard_zone.okta_demo.id
  application_id = keycard_application.api_server.id
  resource_id    = keycard_resource.okta_api.id
}

# Outputs
output "zone_oauth2_redirect_uri" {
  description = "OAuth2 redirect URI - use when configuring OAuth apps"
  value       = keycard_zone.okta_demo.oauth2.redirect_uri
}

output "zone_oauth2_issuer_uri" {
  description = "OAuth2 issuer URI for the zone"
  value       = keycard_zone.okta_demo.oauth2.issuer_uri
}

output "application_client_id" {
  description = "Client ID for the API server application"
  value       = keycard_application_client_secret.api_server.client_id
  sensitive   = true
}

output "application_client_secret" {
  description = "Client secret for the API server application"
  value       = keycard_application_client_secret.api_server.client_secret
  sensitive   = true
}
```

## What Gets Created

### In Keycard

1. **Zone**: A production zone for organizing resources
2. **Identity Provider**: Okta configured for user authentication
3. **Zone User Identity Config**: Links the zone to Okta for user auth
4. **Credential Provider**: Okta configured for credential management
5. **Protected Resource**: A resource requiring Okta credentials
6. **Application**: An API server application
7. **Application Client Secret**: OAuth2 credentials for the application
8. **Application Dependency**: Grants the application access to the protected resource

### In Okta

1. **OAuth App (Identity)**: Used for user authentication flows
2. **OAuth App (Credentials)**: Used for credential issuance
3. **Authorization Server Policy**: Controls access to the credential app
4. **Policy Rule**: Defines allowed grant types and scopes

## Using the Application Credentials

After applying the configuration, retrieve the application credentials:

```bash
terraform output application_client_id
terraform output application_client_secret
```

These credentials can be used by your application to:
1. Authenticate with Keycard
2. Request delegated user credentials for the protected resource
3. Access the Okta-protected API on behalf of users

## Advanced Configuration

### Adding More Resources

To add additional Okta-protected resources:

```hcl
resource "keycard_resource" "another_api" {
  name                   = "Another API"
  identifier             = "https://another-api.example.com"
  zone_id                = keycard_zone.production.id
  credential_provider_id = keycard_provider.okta_credentials.id

  oauth2 = {
    scopes = ["custom:scope"]
  }
}

resource "keycard_application_dependency" "api_server_another" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.api_server.id
  resource_id    = keycard_resource.another_api.id
}
```

### Adding Workload Identity

For Kubernetes-based applications, add workload identity:

```hcl
# First, create an EKS provider
resource "keycard_provider" "eks" {
  zone_id    = keycard_zone.production.id
  name       = "EKS Cluster"
  identifier = "https://oidc.eks.us-east-1.amazonaws.com/id/YOUR-CLUSTER-ID"
}

# Then configure workload identity
resource "keycard_application_workload_identity" "api_server_k8s" {
  zone_id        = keycard_zone.production.id
  application_id = keycard_application.api_server.id
  provider_id    = keycard_provider.eks.id
  subject        = "system:serviceaccount:production:api-server-sa"
}
```

## Security Considerations

1. **Never commit `terraform.tfvars`** - Add it to `.gitignore`
2. **Use environment variables** for sensitive values in CI/CD
3. **Rotate credentials regularly** - Create new `keycard_application_client_secret` resources
4. **Scope permissions carefully** - Only grant access to required resources
5. **Use workload identity** when possible instead of static credentials

## Troubleshooting

### OAuth App Not Found

If you see errors about OAuth apps not being found, ensure:
- Your Okta API token has admin privileges
- The OAuth apps were created successfully in Okta
- You're using the correct Okta organization name

### Authorization Policy Errors

If authorization fails:
- Verify the custom scope exists in your Okta auth server
- Check that the policy rule allows the required grant types
- Ensure the client is whitelisted in the policy

## Clean Up

To remove all created resources:

```bash
terraform destroy
```

This will remove all Keycard and Okta resources created by this configuration.

## Next Steps

- Explore [Application Resources](../../resources/keycard_application/resource.tf)
- Learn about [Resource Configuration](../../resources/keycard_resource/resource.tf)
- Set up [Workload Identity](../../resources/keycard_application_workload_identity/resource.tf)
- Configure [Multiple Providers](../../resources/keycard_provider/resource.tf)
