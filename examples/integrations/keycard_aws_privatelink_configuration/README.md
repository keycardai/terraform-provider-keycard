# Keycard AWS PrivateLink Configuration

This example demonstrates how to configure AWS PrivateLink integration with Keycard in your AWS account. This setup allows your resources to securely communicate with Keycard's services through a private network connection without traversing the public internet.

## What This Example Creates

- **VPC Endpoint**: Creates an interface VPC endpoint that connects to Keycard's VPC Endpoint Service
- **Route53 Private Hosted Zones**: Configures private DNS resolution for three separate domains, each serving a specific purpose:
  - `console.keycard.ai` - For loading and accessing the Keycard web console
  - `api.keycard.ai` - For management API operations (creating applications, managing resources, etc.)
  - `*.keycard.cloud` - Wildcard for zone-specific endpoints that handle your application's runtime operations

## Why Private Hosted Zones?

These Private Hosted Zones are configured to ensure the PrivateLink connection operates seamlessly. By creating these zones and pointing them to the VPC endpoint, all Keycard traffic from your VPC is automatically routed through the private connection instead of the public internet. This provides:

- **Enhanced Security**: Traffic never leaves AWS's private network
- **Simplified Configuration**: Your applications continue using the same Keycard domain names without code changes

## Prerequisites

- **Contact Keycard for Allowlisting**: You must reach out to Keycard with your AWS account ID before creating the VPC endpoint. Keycard needs to allowlist your account on their VPC Endpoint Service. Note that Keycard performs asynchronous approval of connection requests, which is not automatic and may take some time to process.
- An existing VPC with at least one subnet in us-east-1a, us-east-1b or us-east-1c
- Appropriate IAM permissions to create VPC endpoints and Route53 hosted zones
- The VPC must have DNS support and DNS hostnames enabled

## Usage

1. Set the required variables:

```hcl
vpc_id     = "vpc-xxxxxxxxx"
subnet_ids = ["subnet-xxxxxxxxx", "subnet-yyyyyyyyy"]
aws_region = "us-east-1"
```

2. Optionally, provide security group IDs if you need custom security group rules:

```hcl
security_group_ids = ["sg-xxxxxxxxx"]
```

If not provided, the VPC endpoint will use the default VPC security group.

3. Run Terraform:

```bash
terraform init
terraform plan
terraform apply
```

**Note**: After the VPC endpoint is created, it will be in a "pending acceptance" state. Keycard must manually approve the connection request. The approval process is asynchronous and not automatic. Monitor the VPC endpoint state or contact Keycard to confirm when the connection has been approved.

## Security Group Considerations

Ensure that the security group(s) associated with the VPC endpoint allow:
- Inbound HTTPS (port 443) from your application resources
- Outbound traffic as needed

## Verification

After applying this configuration, you can verify the setup:

1. Check that the VPC endpoint is available:
```bash
aws ec2 describe-vpc-endpoints --vpc-endpoint-ids <endpoint-id>
```

2. Test DNS resolution from an EC2 instance in your VPC:
```bash
nslookup api.keycard.ai
```

The DNS name should resolve to the private IP addresses of the VPC endpoint.

## Cost Considerations

- VPC endpoint: Hourly charge + data processing charges
- Route53 private hosted zones: Monthly charge per zone

Refer to AWS pricing documentation for current rates.
