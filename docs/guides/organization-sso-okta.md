# Organization SSO Using Okta

This guide configures Single Sign-On for your Keycard Organization using an existing Okta IDP.

-> **Note:** If you are looking to configure Okta as an identity and/or credential provider for applications within a Keycard Zone, see [Zone Identity Provider Using Okta](zone-identity-provider-okta.md).

## Prerequisites

### 1. Keycard Organization & Service Account

Before using Terraform, you need Keycard API credentials:

1. **Create a Keycard Organization** at [console.keycard.ai](https://console.keycard.ai)
2. **Create a Service Account** in your organization:
   - Go to **Settings** > **Service Accounts**
   - Click **Create Service Account**
   - Copy the **Client ID** and **Client Secret** (you won't see the secret again!)

### 2. Okta Admin Access

You need an Okta API token:

1. Log into your Okta Admin Console
2. Go to **Security** > **API** > **Tokens**
3. Click **Create Token**, give it a name like "Keycard Terraform"
4. Copy the token value

### 3. Okta Group ID

You need the ID of an Okta group whose members should have SSO access:

1. Log into your Okta Admin Console
2. Go to **Directory** > **Groups**
3. Click on the group you want to assign
4. Copy the group ID from the URL (e.g., `00g1234567890abcdef`)

## Quick Start

### 1. Set Up Variables

Create a `terraform.tfvars` file with your configuration:

```hcl
# Keycard credentials
keycard_client_id     = "your-keycard-client-id"
keycard_client_secret = "your-keycard-client-secret"

# Okta credentials
okta_org_name  = "dev-12345"
okta_api_token = "your-okta-api-token"
okta_group_id  = "00g1234567890abcdef"
```

Create a `variables.tf` file with the following variable declarations:

```hcl
variable "keycard_client_id" {
  type        = string
  description = "Keycard service account client ID"
}

variable "keycard_client_secret" {
  type        = string
  sensitive   = true
  description = "Keycard service account client secret"
}

variable "okta_org_name" {
  type        = string
  description = "Okta organization name (the subdomain from your-org.okta.com)"
}

variable "okta_api_token" {
  type        = string
  sensitive   = true
  description = "Okta API token with admin privileges"
}

variable "okta_group_id" {
  type        = string
  description = "Okta group ID to assign to the SSO app"
}

variable "okta_base_url" {
  type        = string
  default     = "okta.com"
  description = "Okta base URL (okta.com for production, oktapreview.com for preview)"
}

variable "keycard_redirect_uri" {
  type        = string
  default     = "https://id.keycard.ai/oauth/2/redirect"
  description = "Keycard OAuth redirect URI for SSO"
}
```

### 2. Apply Terraform

Add a `main.tf` file with the following configuration:

```hcl
terraform {
  required_providers {
    keycard = {
      source  = "keycardai/keycard"
      version = "~> 0.3"
    }
    okta = {
      source  = "okta/okta"
      version = "~> 4.0"
    }
  }
}

provider "keycard" {
  client_id     = var.keycard_client_id
  client_secret = var.keycard_client_secret
}

provider "okta" {
  org_name  = var.okta_org_name
  base_url  = var.okta_base_url
  api_token = var.okta_api_token
}

# Create OIDC app in Okta for SSO
resource "okta_app_oauth" "keycard_sso" {
  label                      = "Keycard SSO"
  type                       = "web"
  grant_types                = ["authorization_code"]
  redirect_uris              = [var.keycard_redirect_uri]
  response_types             = ["code"]
  token_endpoint_auth_method = "client_secret_basic"
}

# Assign group to the app
resource "okta_app_group_assignments" "keycard_sso" {
  app_id = okta_app_oauth.keycard_sso.id
  group {
    id = var.okta_group_id
  }
}

# Configure Keycard to use Okta for SSO
resource "keycard_sso_connection" "okta" {
  identifier    = "https://${var.okta_org_name}.${var.okta_base_url}"
  client_id     = okta_app_oauth.keycard_sso.client_id
  client_secret = okta_app_oauth.keycard_sso.client_secret
}
```

Run `terraform apply -var-file=terraform.tfvars` to deploy.

## What Gets Created

### In Okta

- **OAuth App**: OIDC web app for SSO authentication
- **Group Assignment**: Assigns your specified group to the SSO app

### In Keycard

- **SSO Connection**: Links your organization to Okta for SSO

## After Applying

Once applied, your team can sign into Keycard using their Okta credentials:

1. Go to your Keycard login page
2. Click **Sign in with SSO**
3. Authenticate with Okta

## Cleanup

To remove the SSO configuration:

```bash
terraform destroy
```

This removes the Okta app and disconnects SSO from Keycard. It will revert to only be accessible by the original user(s).
