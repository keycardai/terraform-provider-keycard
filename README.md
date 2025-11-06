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
  endpoint      = "https://api.keycard.ai"  # Optional
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

## Releasing

The provider uses an automated release process powered by [GoReleaser](https://goreleaser.com/) and GitHub Actions. When a version tag is pushed, GitHub Actions automatically builds, signs, and publishes the release.

### Prerequisites

Before creating a release, ensure the following GitHub secrets are configured in the repository:

- `GPG_PRIVATE_KEY` - GPG private key for signing release artifacts
- `PASSPHRASE` - Passphrase for the GPG private key
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

### Release Steps

1. **Update the Changelog**

   Edit [`CHANGELOG.md`](./CHANGELOG.md) to document all changes in the new version. Follow the existing format with sections for FEATURES, RESOURCES, DATA SOURCES, and DOCUMENTATION.

   ```markdown
   ## [x.y.z] - YYYY-MM-DD

   ### FEATURES
   - New feature description

   ### RESOURCES
   - Resource changes

   ### DATA SOURCES
   - Data source changes
   ```

2. **Create and Push a Version Tag**

   Create a git tag following semantic versioning (e.g., `v1.0.0`, `v0.2.1`):

   ```bash
   # Create an annotated tag
   git tag -a v0.2.0 -m "Release v0.2.0"

   # Push the tag to GitHub
   git push origin v0.2.0
   ```

3. **Monitor the Release**

   GitHub Actions will automatically:
   - Build binaries for multiple platforms (Linux, macOS, Windows, FreeBSD)
   - Build for multiple architectures (amd64, 386, arm, arm64)
   - Generate SHA256 checksums
   - Sign checksums with GPG
   - Create a GitHub release with all artifacts
   - Include `terraform-registry-manifest.json` for registry compatibility

   Monitor the release workflow at: `https://github.com/keycardai/terraform-provider-keycard/actions`

### Terraform Registry

After a successful GitHub release, the Terraform Registry should automatically detect and publish the new version. This typically happens within a few minutes of the release being created.

Verify the new version appears at: `https://registry.terraform.io/providers/keycardai/keycard/latest`

## Contributing

This provider is maintained internally by the Keycard team. **We do not accept external contributions at this time.**

If you encounter issues or have questions, please refer to the [official documentation](https://docs.keycard.ai) or contact Keycard support.

## License

Mozilla Public License 2.0 - see [LICENSE](LICENSE) for details.
