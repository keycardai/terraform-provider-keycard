data "aws_caller_identity" "current" {}

# Retrieve a valid KMS key policy for use with Keycard's bring your own encryption support
data "keycard_key_policy" "this" {
  account_id = data.aws_caller_identity.current.account_id
}

# Print out the rendered policy JSON
output "keycard_key_policy" {
  value = data.keycard_key_policy.this
}

# Create a KMS key which can be used for BYOK
resource "aws_kms_key" "encryption" {
  description = "KMS key used for Keycard encryption"
  policy      = data.keycard_key_policy.this.policy
}
