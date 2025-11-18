# Fetch a KMS key policy for customer-managed KMS encryption keys
# This policy can be used when creating AWS KMS keys for Keycard zone encryption
data "keycard_aws_kms_key_policy" "example_policy" {
  # AWS account ID the KMS key belongs to. The root user in the account
  # will be granted access to the key in the policy so you do not lose access
  # to the key when applying this policy verbatim.
  account_id = "123456789012"
}

# Use the policy when creating an AWS KMS key
# This example shows how to integrate with the AWS provider
resource "aws_kms_key" "example_key" {
  description = "KMS key for Keycard zone encryption"
  policy      = data.keycard_aws_kms_key_policy.example_policy.policy
}

# Create a zone using the key that was just created
resource "keycard_zone" "example_zone" {
  name        = "My Zone"
  description = "Zone with data encrypted using customer managed KMS key"

  encryption_key {
    aws {
      arn = aws_kms_key.example_key.arn
    }
  }
}

# Example: Combining the Keycard policy with additional custom policies
# This is useful when you need to grant additional permissions beyond what
# Keycard requires, such as allowing other AWS services or principals to use the key

# Define additional custom policy statements
data "aws_iam_policy_document" "additional_kms_permissions" {
  statement {
    sid    = "AllowCloudWatchLogs"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["logs.amazonaws.com"]
    }

    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:CreateGrant",
      "kms:DescribeKey"
    ]

    resources = ["*"]

    condition {
      test     = "ArnLike"
      variable = "kms:EncryptionContext:aws:logs:arn"
      values   = ["arn:aws:logs:*:123456789012:log-group:*"]
    }
  }
}

# Combine the Keycard policy with your additional policies
# The source_policy_documents attribute merges multiple policy documents
data "aws_iam_policy_document" "combined_kms_policy" {
  source_policy_documents = [
    data.keycard_aws_kms_key_policy.example_policy.policy,
    data.aws_iam_policy_document.additional_kms_permissions.json
  ]
}

# Create a KMS key with the combined policy
resource "aws_kms_key" "combined_policy_key" {
  description = "KMS key for Keycard with additional CloudWatch Logs permissions"
  policy      = data.aws_iam_policy_document.combined_kms_policy.json
}

# Create a zone using the key with combined policy
resource "keycard_zone" "combined_policy_zone" {
  name        = "My Zone with Combined Policy"
  description = "Zone with customer managed KMS key that has additional permissions"

  encryption_key {
    aws {
      arn = aws_kms_key.combined_policy_key.arn
    }
  }
}
