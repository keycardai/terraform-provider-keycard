# MCP Server Terraform Module

This Terraform module creates the necessary Keycard resources to set up a Model Context Protocol (MCP) server with proper authentication and authorization.

## What This Module Creates

- **Application**: Represents the MCP server in Keycard
- **Resource**: Represents the MCP server as an API that can be accessed by other applications
- **Client Secret**: OAuth2 credentials for the MCP server to authenticate with Keycard

## Required Inputs

- `zone_id`: The ID of the Keycard zone where resources will be created
- `mcp_server_url`: The URL of your MCP server (used as both application and resource identifier)

## Basic Usage

```hcl
module "mcp_server" {
  source = "./templates/mcp-server"

  zone_id          = "zone-12345"
  mcp_server_url   = "https://my-mcp-server.example.com"
  application_name = "My MCP Server"
  env_file_name    = "mcp-server.env"
}
```

## Advanced Usage with OAuth2 Scopes

```hcl
# MCP server that provides access to APIs with specific scopes
module "api_mcp_server" {
  source = "./templates/mcp-server"

  zone_id                = "zone-12345"
  mcp_server_url         = "https://api-mcp.example.com"
  application_name       = "API MCP Server"
  application_description = "MCP server providing access to various APIs"
  resource_name          = "API MCP Server"
  env_file_name          = "api-mcp-server.env"

  # Define OAuth2 scopes for the MCP server API
  oauth2_scopes = [
    "read:calendar",
    "write:documents",
    "read:profile"
  ]
}
```

## Usage with OAuth2 Redirect URIs

If your MCP server needs OAuth2 callbacks (e.g., for web-based authentication):

```hcl
module "web_mcp_server" {
  source = "./templates/mcp-server"

  zone_id         = "zone-12345"
  mcp_server_url  = "https://web-mcp.example.com"
  application_name = "Web MCP Server"

  oauth2_redirect_uris = [
    "https://web-mcp.example.com/oauth/callback"
  ]
}
```

## Accessing the Generated Credentials

The module outputs OAuth2 credentials that your MCP server can use:

```hcl
# Output the credentials for use in your MCP server configuration
output "mcp_client_id" {
  description = "Client ID for MCP server authentication"
  value       = module.mcp_server.client_id
  sensitive   = true
}

output "mcp_client_secret" {
  description = "Client secret for MCP server authentication"
  value       = module.mcp_server.client_secret
  sensitive   = true
}
```

## Storing Credentials Securely

### AWS Secrets Manager

```hcl
resource "aws_secretsmanager_secret" "mcp_credentials" {
  name        = "mcp-server-oauth-credentials"
  description = "OAuth2 credentials for MCP Server"
}

resource "aws_secretsmanager_secret_version" "mcp_credentials" {
  secret_id     = aws_secretsmanager_secret.mcp_credentials.id
  secret_string = module.mcp_server.oauth2_credentials
}
```

### Kubernetes Secret

```hcl
resource "kubernetes_secret" "mcp_credentials" {
  metadata {
    name      = "mcp-server-credentials"
    namespace = "default"
  }

  data = {
    client_id     = module.mcp_server.client_id
    client_secret = module.mcp_server.client_secret
  }

  type = "Opaque"
}
```

## Input Variables

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| zone_id | The ID of the Keycard zone | `string` | n/a | yes |
| mcp_server_url | The URL of the MCP server | `string` | n/a | yes |
| application_name | The name of the MCP server application | `string` | `"MCP Server"` | no |
| application_description | Description of the MCP server application | `string` | `"Model Context Protocol server for API access"` | no |
| resource_name | The name of the MCP server resource | `string` | `"MCP Server API"` | no |
| oauth2_scopes | List of OAuth2 scopes for the resource | `list(string)` | `[]` | no |
| application_traits | List of traits to apply to the application | `list(string)` | `[]` | no |
| oauth2_redirect_uris | List of OAuth2 redirect URIs for the application | `list(string)` | `[]` | no |
| env_file_name | Name of the environment file to generate | `string` | `".env"` | no |

## Output Values

| Name | Description | Sensitive |
|------|-------------|:---------:|
| application_id | The ID of the created MCP server application | no |
| application_name | The name of the MCP server application | no |
| application_identifier | The identifier (URL) of the MCP server application | no |
| client_id | OAuth2 client ID for the MCP server | yes |
| client_secret | OAuth2 client secret for the MCP server | yes |
| resource_id | The ID of the MCP server resource | no |
| resource_name | The name of the MCP server resource | no |
| resource_identifier | The identifier of the MCP server resource | no |
| oauth2_credentials | OAuth2 credentials as JSON | yes |


## Examples

See the `examples/` directory for complete working examples of different MCP server configurations.
