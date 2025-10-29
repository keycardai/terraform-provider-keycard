# Terraform Provider for Keycard

This is the official Terraform provider for [Keycard](https://keycard.ai), enabling you to manage Keycard resources such as zones, applications, providers, and identity configurations using Terraform.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.7
- [Go](https://golang.org/doc/install) >= 1.24 (for development)

## Installation

### From Terraform Registry (Production Use)

The provider is published to the Terraform Registry and can be used by declaring it in your Terraform configuration:

```hcl
terraform {
  required_providers {
    keycard = {
      source  = "keycardai/keycard"
      version = "~> 0.1"
    }
  }
}

provider "keycard" {
  client_id     = var.keycard_client_id
  client_secret = var.keycard_client_secret
}
```

### Local Development Installation

For local development and testing, you can build and install the provider locally with development overrides.

#### Step 1: Build and Install the Provider

```bash
# Clone the repository
git clone https://github.com/keycardai/terraform-provider-keycard.git
cd terraform-provider-keycard

# Build and install the provider to $GOPATH/bin
make install
```

This will compile the provider binary and place it in your `$GOPATH/bin` directory (typically `~/go/bin`).

#### Step 2: Configure Terraform Development Overrides

Create or edit `~/.terraformrc` to tell Terraform to use your local build instead of downloading from the registry:

```hcl
provider_installation {
  dev_overrides {
    "keycardai/keycard" = "/Users/YOUR_USERNAME/go/bin"  # Replace with your actual $GOPATH/bin
  }

  # For all other providers, use the registry
  direct {}
}
```

**Important:** Replace `/Users/YOUR_USERNAME/go/bin` with your actual `$GOPATH/bin` directory. You can find it by running `go env GOPATH`.

#### Step 3: Test Your Local Provider

With dev overrides configured, Terraform will use your local build:

```bash
cd examples/
terraform init
terraform plan
terraform apply
```

Note: When using dev overrides, Terraform will display a warning that the provider is being overridden. This is expected behavior.

#### Step 4: Return to Production Provider

When you're done with local development, remove or comment out the `dev_overrides` block in `~/.terraformrc` to use the published provider from the registry again.

## Authentication

The provider requires OAuth2 client credentials for authentication with the Keycard API. You can configure these credentials in three ways:

### 1. Provider Block (not recommended for production)

```hcl
provider "keycard" {
  client_id     = "your-client-id"
  client_secret = "your-client-secret"
  endpoint      = "https://api.keycard.ai"  # Optional, defaults to production API
}
```

### 2. Environment Variables (recommended)

```bash
export KEYCARD_CLIENT_ID="your-client-id"
export KEYCARD_CLIENT_SECRET="your-client-secret"
export KEYCARD_ENDPOINT="https://api.keycard.ai"  # Optional
```

### 3. Terraform Variables

```hcl
variable "keycard_client_id" {
  type      = string
  sensitive = true
}

variable "keycard_client_secret" {
  type      = string
  sensitive = true
}

provider "keycard" {
  client_id     = var.keycard_client_id
  client_secret = var.keycard_client_secret
}
```

## Documentation

Comprehensive documentation for all resources, data sources, and configuration options is available in the [`docs/`](./docs) directory:

- **Provider Configuration:** [`docs/index.md`](./docs/index.md)
- **Resources:** [`docs/resources/`](./docs/resources)
- **Data Sources:** [`docs/data-sources/`](./docs/data-sources)

You can also view the documentation on the [Terraform Registry](https://registry.terraform.io/providers/keycardai/keycard/latest/docs).

## Development

### Building the Provider

```bash
# Build the provider binary
make build

# Build and install to $GOPATH/bin
make install
```

### Running Tests

```bash
# Run unit tests
make test

# Run acceptance tests (creates real resources)
make testacc
```

**Note:** Acceptance tests interact with the real Keycard API and may incur costs or modify resources. Set up appropriate test credentials before running.

### Generating Documentation

Documentation is automatically generated from code comments and example files:

```bash
# Generate provider documentation
make generate
```

This runs `terraform-plugin-docs` to generate documentation from:
- Schema descriptions in the code
- Example Terraform configurations in `examples/`
- Templates in `templates/` (if present)

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint
```

## Contributing

This provider is maintained internally by the Keycard team. **We do not accept external contributions at this time.**

If you encounter issues or have questions, please refer to the [official documentation](https://docs.keycard.ai) or contact Keycard support.

## License

Mozilla Public License 2.0 - see [LICENSE](LICENSE) for details.
