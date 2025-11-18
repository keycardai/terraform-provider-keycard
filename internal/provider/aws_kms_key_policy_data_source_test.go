package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAwsKmsKeyPolicyDataSource_basic(t *testing.T) {
	accountID := "123456789012"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Fetch the KMS key policy with a valid AWS account ID
			{
				Config: testAccAwsKmsKeyPolicyDataSourceConfig_basic(accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the policy attribute is set
					resource.TestCheckResourceAttrSet("data.keycard_aws_kms_key_policy.test", "policy"),
					// Verify the account_id matches what we provided
					resource.TestCheckResourceAttr("data.keycard_aws_kms_key_policy.test", "account_id", accountID),
					// Verify the policy contains the account ID (should have replaced the placeholder)
					resource.TestMatchResourceAttr(
						"data.keycard_aws_kms_key_policy.test",
						"policy",
						regexp.MustCompile(accountID),
					),
					// Verify the policy looks like valid JSON (contains expected AWS policy structure)
					resource.TestMatchResourceAttr(
						"data.keycard_aws_kms_key_policy.test",
						"policy",
						regexp.MustCompile(`"Version"`),
					),
					resource.TestMatchResourceAttr(
						"data.keycard_aws_kms_key_policy.test",
						"policy",
						regexp.MustCompile(`"Statement"`),
					),
				),
			},
		},
	})
}

func TestAccAwsKmsKeyPolicyDataSource_invalidAccountId(t *testing.T) {
	// Test with various invalid account ID formats
	invalidAccountIDs := []string{
		"invalid",           // Not a number
		"12345",             // Too short
		"12345678901234567", // Too long
		"abcdefghijkl",      // Letters
		"123-456-789-012",   // Contains dashes
	}

	for _, accountID := range invalidAccountIDs {
		t.Run(fmt.Sprintf("accountID_%s", accountID), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccAwsKmsKeyPolicyDataSourceConfig_basic(accountID),
						// Validator should reject invalid account ID formats
						ExpectError: regexp.MustCompile(`must be a 12-digit AWS account ID`),
					},
				},
			})
		})
	}
}

func testAccAwsKmsKeyPolicyDataSourceConfig_basic(accountID string) string {
	return fmt.Sprintf(`
data "keycard_aws_kms_key_policy" "test" {
  account_id = %[1]q
}
`, accountID)
}
