package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSOConnectionResource_basic(t *testing.T) {
	identifier := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	clientID := acctest.RandomWithPrefix("client")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSSOConnectionResourceConfig_basic(identifier, clientID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "identifier", identifier),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_id", clientID),
					resource.TestCheckResourceAttrSet("keycard_sso_connection.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "keycard_sso_connection.test",
				ImportState:             true,
				ImportStateVerify:       true,
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

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial values
			{
				Config: testAccSSOConnectionResourceConfig_basic(identifier1, clientID1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "identifier", identifier1),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_id", clientID1),
				),
			},
			// Update identifier and client_id
			{
				Config: testAccSSOConnectionResourceConfig_basic(identifier2, clientID2),
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

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with client_secret
			{
				Config: testAccSSOConnectionResourceConfig_withClientSecret(identifier, clientID, clientSecret),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "identifier", identifier),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_id", clientID),
					resource.TestCheckResourceAttr("keycard_sso_connection.test", "client_secret", clientSecret),
				),
			},
			// ImportState - client_secret should not be imported
			{
				ResourceName:            "keycard_sso_connection.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
		},
	})
}

func TestAccSSOConnectionResource_emptyClientIdInvalid(t *testing.T) {
	identifier := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSSOConnectionResourceConfig_basic(identifier, ""),
				ExpectError: regexp.MustCompile(`Attribute client_id string length must be at least 1`),
			},
		},
	})
}

func TestAccSSOConnectionResource_emptyIdentifierInvalid(t *testing.T) {
	clientID := acctest.RandomWithPrefix("client")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSSOConnectionResourceConfig_basic("", clientID),
				ExpectError: regexp.MustCompile(`Attribute identifier string length must be at least 1`),
			},
		},
	})
}

func TestAccSSOConnectionResource_emptyClientSecretInvalid(t *testing.T) {
	identifier := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	clientID := acctest.RandomWithPrefix("client")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSSOConnectionResourceConfig_withClientSecret(identifier, clientID, ""),
				ExpectError: regexp.MustCompile(`Attribute client_secret string length must be at least 1`),
			},
		},
	})
}

func testAccSSOConnectionResourceConfig_basic(identifier, clientID string) string {
	return fmt.Sprintf(`
resource "keycard_sso_connection" "test" {
  identifier = %[1]q
  client_id  = %[2]q
}
`, identifier, clientID)
}

func testAccSSOConnectionResourceConfig_withClientSecret(identifier, clientID, clientSecret string) string {
	return fmt.Sprintf(`
resource "keycard_sso_connection" "test" {
  identifier    = %[1]q
  client_id     = %[2]q
  client_secret = %[3]q
}
`, identifier, clientID, clientSecret)
}
