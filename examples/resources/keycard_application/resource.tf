# Example zone to create providers in
resource "keycard_zone" "dev" {
  name = "Development"
}

# Google MCP Server application
# MCP servers typically access multiple APIs on behalf of users
resource "keycard_application" "google_mcp_server" {
  name       = "Google MCP Server"
  identifier = "https://google-mcp.example.com"
  zone_id    = keycard_zone.dev.id
}

# Okta MCP Server with OAuth2 redirect URIs
# Web-based MCP servers may need OAuth2 callbacks
resource "keycard_application" "okta_mcp_server" {
  name        = "Okta MCP Server"
  identifier  = "https://okta-mcp.example.com"
  zone_id     = keycard_zone.dev.id
  description = "MCP server for Okta API access"

  oauth2 {
    redirect_uris = [
      "https://okta-mcp.example.com/oauth/callback"
    ]
  }
}

# Backend service application
# Typically accesses resources using service credentials
resource "keycard_application" "backend_service" {
  name        = "Backend Service"
  identifier  = "https://backend.example.com"
  zone_id     = keycard_zone.dev.id
  description = "Internal backend service"
}

# Mobile application with multiple redirect URIs
# Mobile apps often need custom URI schemes and web callbacks
resource "keycard_application" "mobile_app" {
  name        = "Mobile Application"
  identifier  = "com.example.mobile"
  zone_id     = keycard_zone.dev.id
  description = "iOS and Android mobile application"

  oauth2 {
    redirect_uris = [
      "com.example.mobile://oauth/callback",
      "https://app.example.com/mobile/callback"
    ]
  }
}

# API Gateway application
# Gateways route requests and manage access to downstream services
resource "keycard_application" "api_gateway" {
  name        = "API Gateway"
  identifier  = "https://api.example.com"
  zone_id     = keycard_zone.dev.id
  description = "Central API gateway for all services"

  # The gateway trait enables gateway-specific behaviors and workflows
  traits = ["gateway"]
}
