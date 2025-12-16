package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSSOConnectionResource_basic(t *testing.T) {
	identifier := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	clientID := acctest.RandomWithPrefix("client")
	orgID := os.Getenv("KEYCARD_TEST_ORGANIZATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckSSOConnection(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSSOConnectionResourceConfig_basic(orgID, identifier, clientID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "organization_id", orgID),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "identifier", identifier),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_id", clientID),
					resource.TestCheckResourceAttrSet("keycard_sso_connection.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_sso_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs := state.RootModule().Resources["keycard_sso_connection.test"]
					return fmt.Sprintf("organizations/%s/sso-connection", rs.Primary.Attributes["organization_id"]), nil
				},
				// client_secret is write-only, won't match on import
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSSOConnectionResource_update(t *testing.T) {
	identifier1 := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	identifier2 := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	clientID1 := acctest.RandomWithPrefix("client1")
	clientID2 := acctest.RandomWithPrefix("client2")
	orgID := os.Getenv("KEYCARD_TEST_ORGANIZATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckSSOConnection(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial values
			{
				Config: testAccSSOConnectionResourceConfig_basic(orgID, identifier1, clientID1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "identifier", identifier1),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_id", clientID1),
				),
			},
			// Update identifier and client_id
			{
				Config: testAccSSOConnectionResourceConfig_basic(orgID, identifier2, clientID2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "identifier", identifier2),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_id", clientID2),
				),
			},
		},
	})
}

func TestAccSSOConnectionResource_withClientSecret(t *testing.T) {
	identifier := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	clientID := acctest.RandomWithPrefix("client")
	clientSecret := acctest.RandomWithPrefix("secret")
	orgID := os.Getenv("KEYCARD_TEST_ORGANIZATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckSSOConnection(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with client_secret
			{
				Config: testAccSSOConnectionResourceConfig_withClientSecret(orgID, identifier, clientID, clientSecret),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "identifier", identifier),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_id", clientID),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_secret", clientSecret),
				),
			},
			// ImportState - client_secret should not be imported
			{
				ResourceName:      "keycard_sso_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs := state.RootModule().Resources["keycard_sso_connection.test"]
					return fmt.Sprintf("organizations/%s/sso-connection", rs.Primary.Attributes["organization_id"]), nil
				},
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
		},
	})
}

func TestAccSSOConnectionResource_emptyClientIdInvalid(t *testing.T) {
	identifier := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	orgID := os.Getenv("KEYCARD_TEST_ORGANIZATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckSSOConnection(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSSOConnectionResourceConfig_basic(orgID, identifier, ""),
				ExpectError: regexp.MustCompile(`Attribute client_id string length must be at least 1`),
			},
		},
	})
}

func TestAccSSOConnectionResource_emptyIdentifierInvalid(t *testing.T) {
	clientID := acctest.RandomWithPrefix("client")
	orgID := os.Getenv("KEYCARD_TEST_ORGANIZATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckSSOConnection(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSSOConnectionResourceConfig_basic(orgID, "", clientID),
				ExpectError: regexp.MustCompile(`Attribute identifier string length must be at least 1`),
			},
		},
	})
}

func TestAccSSOConnectionResource_emptyClientSecretInvalid(t *testing.T) {
	identifier := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	clientID := acctest.RandomWithPrefix("client")
	orgID := os.Getenv("KEYCARD_TEST_ORGANIZATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckSSOConnection(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSSOConnectionResourceConfig_withClientSecret(orgID, identifier, clientID, ""),
				ExpectError: regexp.MustCompile(`Attribute client_secret string length must be at least 1`),
			},
		},
	})
}

// testAccPreCheckSSOConnection checks that the KEYCARD_TEST_ORGANIZATION_ID env var is set.
func testAccPreCheckSSOConnection(t *testing.T) {
	if os.Getenv("KEYCARD_TEST_ORGANIZATION_ID") == "" {
		t.Skip("KEYCARD_TEST_ORGANIZATION_ID must be set for SSO connection acceptance tests")
	}
}

func testAccSSOConnectionResourceConfig_basic(orgID, identifier, clientID string) string {
	return fmt.Sprintf(`
resource "keycard_sso_connection" "test" {
  organization_id = %[1]q
  identifier      = %[2]q
  client_id       = %[3]q
}
`, orgID, identifier, clientID)
}

func testAccSSOConnectionResourceConfig_withClientSecret(orgID, identifier, clientID, clientSecret string) string {
	return fmt.Sprintf(`
resource "keycard_sso_connection" "test" {
  organization_id = %[1]q
  identifier      = %[2]q
  client_id       = %[3]q
  client_secret   = %[4]q
}
`, orgID, identifier, clientID, clientSecret)
}
