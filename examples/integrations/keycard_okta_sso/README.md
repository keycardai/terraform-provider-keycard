# Keycard + Okta SSO Bootstrap

This example configures Okta as the SSO provider for your Keycard organization in a single `terraform apply`.

## Prerequisites

### 1. Keycard Organization & Service Account

Before using Terraform, you need Keycard API credentials:

1. **Create a Keycard Organization** at [app.keycard.ai](https://app.keycard.ai) (or your self-hosted instance)
2. **Create a Service Account** in your organization:
   - Go to **Settings** → **Service Accounts**
   - Click **Create Service Account**
   - Copy the **Client ID** and **Client Secret** (you won't see the secret again!)

These credentials (`keycard_client_id` and `keycard_client_secret`) are what Terraform uses to manage your org.

### 2. Okta Admin Access

You need an Okta API token:

1. Log into your Okta Admin Console
2. Go to **Security** → **API** → **Tokens**
3. Click **Create Token**, give it a name like "Keycard Terraform"
4. Copy the token value

## Usage

```bash
# Copy the example vars file
cp terraform.tfvars.example terraform.tfvars

# Edit with your credentials
vim terraform.tfvars

# Initialize and apply
terraform init
terraform apply
```

## What Gets Created

| Resource | Location | Description |
|----------|----------|-------------|
| `okta_app_oauth.keycard_sso` | Okta | OIDC web app for SSO authentication |
| `okta_app_group_assignments` | Okta | Assigns all users to the SSO app |
| `keycard_sso_connection.okta` | Keycard | Links your org to Okta for SSO |

## After Applying

Once applied, your team can sign into Keycard using their Okta credentials:

1. Go to your Keycard login page
2. Click **Sign in with SSO**
3. Authenticate with Okta
4. You're in!

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `keycard_client_id` | Yes | - | Service account client ID |
| `keycard_client_secret` | Yes | - | Service account client secret |
| `keycard_endpoint` | No | `https://api.keycard.ai` | API endpoint |
| `organization_id` | Yes | - | Your Keycard org ID |
| `keycard_redirect_uri` | No | `http://id.keycard.ai/oauth/2/redirect` | OAuth callback URL |
| `okta_org_name` | Yes | - | Okta subdomain (from `xxx.okta.com`) |
| `okta_base_url` | No | `okta.com` | Okta domain (`oktapreview.com` for preview orgs) |
| `okta_api_token` | Yes | - | Okta admin API token |

## Cleanup

To remove the SSO configuration:

```bash
terraform destroy
```

This removes the Okta app and disconnects SSO from Keycard. Users will need to use other authentication methods.
