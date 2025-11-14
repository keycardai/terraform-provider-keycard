package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccKeyPolicyDataSource_basic tests fetching a KMS key policy with a valid AWS account ID
func TestAccKeyPolicyDataSource_basic(t *testing.T) {
	// Use a test AWS account ID
	accountID := "123456789012"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeyPolicyDataSourceConfig_basic(accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify account_id is set correctly
					resource.TestCheckResourceAttr("data.keycard_key_policy.test", "account_id", accountID),
					// Verify policy is returned and not empty
					resource.TestCheckResourceAttrSet("data.keycard_key_policy.test", "policy"),
					// Verify policy contains the account ID (not the placeholder)
					resource.TestMatchResourceAttr("data.keycard_key_policy.test", "policy", regexp.MustCompile(accountID)),
					// Verify policy does not contain the placeholder
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.keycard_key_policy.test"]
						if !ok {
							return nil
						}
						policy := rs.Primary.Attributes["policy"]
						if regexp.MustCompile("<YOUR_AWS_ACCOUNT_ID>").MatchString(policy) {
							t.Error("Policy still contains placeholder <YOUR_AWS_ACCOUNT_ID>")
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccKeyPolicyDataSource_differentAccountID tests with a different account ID
func TestAccKeyPolicyDataSource_differentAccountID(t *testing.T) {
	accountID := "999888777666"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeyPolicyDataSourceConfig_basic(accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_key_policy.test", "account_id", accountID),
					resource.TestCheckResourceAttrSet("data.keycard_key_policy.test", "policy"),
					resource.TestMatchResourceAttr("data.keycard_key_policy.test", "policy", regexp.MustCompile(accountID)),
				),
			},
		},
	})
}

// TestAccKeyPolicyDataSource_invalidAccountID tests validation for invalid account IDs
func TestAccKeyPolicyDataSource_invalidAccountID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test with too short account ID
			{
				Config:      testAccKeyPolicyDataSourceConfig_basic("12345"),
				ExpectError: regexp.MustCompile("Attribute account_id string length must be at least 12"),
			},
			// Test with too long account ID
			{
				Config:      testAccKeyPolicyDataSourceConfig_basic("1234567890123"),
				ExpectError: regexp.MustCompile("Attribute account_id string length must be at most 12"),
			},
		},
	})
}

// TestAccKeyPolicyDataSource_withRealKMSKey tests the data source with environment-provided KMS key
func TestAccKeyPolicyDataSource_withRealKMSKey(t *testing.T) {
	accountID := os.Getenv("KEYCARD_TEST_AWS_ACCOUNT_ID")
	if accountID == "" {
		t.Skip("KEYCARD_TEST_AWS_ACCOUNT_ID not set, skipping test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeyPolicyDataSourceConfig_basic(accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_key_policy.test", "account_id", accountID),
					resource.TestCheckResourceAttrSet("data.keycard_key_policy.test", "policy"),
					// Verify the policy is valid JSON
					resource.TestMatchResourceAttr("data.keycard_key_policy.test", "policy", regexp.MustCompile(`^\{.*\}$`)),
					// Verify policy contains expected KMS policy structure
					resource.TestMatchResourceAttr("data.keycard_key_policy.test", "policy", regexp.MustCompile(`"Statement"`)),
					resource.TestMatchResourceAttr("data.keycard_key_policy.test", "policy", regexp.MustCompile(`"Principal"`)),
				),
			},
		},
	})
}

// Helper function to generate test configuration
func testAccKeyPolicyDataSourceConfig_basic(accountID string) string {
	return `
data "keycard_key_policy" "test" {
  account_id = "` + accountID + `"
}
`
}

// Unit test for replaceAccountIDPlaceholder function
func TestReplaceAccountIDPlaceholder(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		accountID string
		expected  string
	}{
		{
			name:      "single occurrence",
			input:     "arn:aws:iam::<YOUR_AWS_ACCOUNT_ID>:root",
			accountID: "123456789012",
			expected:  "arn:aws:iam::123456789012:root",
		},
		{
			name:      "multiple occurrences",
			input:     "arn:aws:iam::<YOUR_AWS_ACCOUNT_ID>:user/<YOUR_AWS_ACCOUNT_ID>",
			accountID: "999888777666",
			expected:  "arn:aws:iam::999888777666:user/999888777666",
		},
		{
			name:      "no placeholder",
			input:     "arn:aws:iam::123456789012:root",
			accountID: "999888777666",
			expected:  "arn:aws:iam::123456789012:root",
		},
		{
			name:      "empty string",
			input:     "",
			accountID: "123456789012",
			expected:  "",
		},
		{
			name:      "placeholder only",
			input:     "<YOUR_AWS_ACCOUNT_ID>",
			accountID: "123456789012",
			expected:  "123456789012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceAccountIDPlaceholder(tt.input, tt.accountID)
			if result != tt.expected {
				t.Errorf("replaceAccountIDPlaceholder(%q, %q) = %q, want %q",
					tt.input, tt.accountID, result, tt.expected)
			}
		})
	}
}
