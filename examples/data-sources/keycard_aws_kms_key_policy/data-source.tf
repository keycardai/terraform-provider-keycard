# Fetch a KMS key policy for customer-managed KMS encryption keys
# This policy can be used when creating AWS KMS keys for Keycard zone encryption
data "keycard_aws_kms_key_policy" "example" {
  account_id = "123456789012"
}

# Use the policy when creating an AWS KMS key
# This example shows how to integrate with the AWS provider
resource "aws_kms_key" "keycard_zone" {
  description = "KMS key for Keycard zone encryption"
  policy      = data.keycard_aws_kms_key_policy.example.policy
}

# Output the policy for reference
output "kms_key_policy" {
  description = "KMS key policy document for Keycard"
  value       = data.keycard_aws_kms_key_policy.example.policy
}
