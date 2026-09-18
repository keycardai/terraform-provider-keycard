variable "zone_id" {
  description = "The ID of the Keycard zone where the MCP server will be created"
  type        = string
}

variable "mcp_server_url" {
  description = "The URL of the MCP server (used as both application and resource identifier)"
  type        = string
}

variable "application_name" {
  description = "The name of the MCP server application"
  type        = string
  default     = "MCP Server"
}

variable "application_description" {
  description = "Description of the MCP server application"
  type        = string
  default     = "Model Context Protocol server for API access"
}

variable "resource_name" {
  description = "The name of the MCP server resource"
  type        = string
  default     = "MCP Server API"
}

variable "env_file_name" {
  description = "Name of the environment file to generate with Keycard configuration"
  type        = string
  default     = ".env"
}
