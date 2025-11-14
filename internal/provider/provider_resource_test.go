package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccProviderResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProviderResourceConfig_basic(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
					resource.TestCheckResourceAttrSet("keycard_provider.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_provider.test", "zone_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_provider.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs := state.RootModule().Resources["keycard_provider.test"]
					return fmt.Sprintf("zones/%s/providers/%s", rs.Primary.Attributes["zone_id"], rs.Primary.ID), nil
				},
				// client_secret is not returned by API, so it won't match on import
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			// Update and Read testing
			{
				Config: testAccProviderResourceConfig_basic(rName+"-updated", identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccProviderResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccProviderResourceConfig_withDescription(rName, identifier, "Test description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_provider.test", "description", "Test description"),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
				),
			},
			// Update description
			{
				Config: testAccProviderResourceConfig_withDescription(rName, identifier, "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "description", "Updated description"),
				),
			},
			// Remove description
			{
				Config: testAccProviderResourceConfig_basic(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_provider.test", "description"),
				),
			},
		},
	})
}

func TestAccProviderResource_oauth2Config(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with full OAuth2 configuration
			{
				Config: testAccProviderResourceConfig_oauth2Config(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
					resource.TestCheckResourceAttr("keycard_provider.test", "client_id", "test-client-id"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.authorization_endpoint", identifier+"/authorize"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.token_endpoint", identifier+"/token"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_provider.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs := state.RootModule().Resources["keycard_provider.test"]
					return fmt.Sprintf("zones/%s/providers/%s", rs.Primary.Attributes["zone_id"], rs.Primary.ID), nil
				},
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
		},
	})
}

func TestAccProviderResource_oauth2Updates(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with basic config
			{
				Config: testAccProviderResourceConfig_basic(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
				),
			},
			// Update to add client_id and oauth2 endpoints
			{
				Config: testAccProviderResourceConfig_oauth2Config(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
					resource.TestCheckResourceAttr("keycard_provider.test", "client_id", "test-client-id"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.authorization_endpoint", identifier+"/authorize"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.token_endpoint", identifier+"/token"),
				),
			},
			// Update back to basic (remove optional fields)
			{
				Config: testAccProviderResourceConfig_basic(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
					resource.TestCheckNoResourceAttr("keycard_provider.test", "client_id"),
					resource.TestCheckResourceAttrSet("keycard_provider.test", "oauth2.authorization_endpoint"),
					resource.TestCheckResourceAttrSet("keycard_provider.test", "oauth2.token_endpoint"),
				),
			},
		},
	})
}

func TestAccProviderResource_emptyDescriptionInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withDescription(rName, identifier, ""),
				ExpectError: regexp.MustCompile(`Attribute description string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_emptyClientIdInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withClientId(rName, identifier, ""),
				ExpectError: regexp.MustCompile(`Attribute client_id string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_emptyClientSecretInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withClientSecret(rName, identifier, ""),
				ExpectError: regexp.MustCompile(`Attribute client_secret string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_emptyAuthorizationEndpointInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withOAuth2Endpoints(rName, identifier, "", "https://token.example.com"),
				ExpectError: regexp.MustCompile(`Attribute oauth2.authorization_endpoint string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_emptyTokenEndpointInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withOAuth2Endpoints(rName, identifier, "https://auth.example.com", ""),
				ExpectError: regexp.MustCompile(`Attribute oauth2.token_endpoint string length must be at least 1`),
			},
		},
	})
}

func testAccProviderResourceConfig_basic(name, identifier string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[1]q
  zone_id    = keycard_zone.test.id
  identifier = %[2]q
}
`, name, identifier)
}

func testAccProviderResourceConfig_withDescription(name, identifier, description string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name        = %[1]q
  zone_id     = keycard_zone.test.id
  identifier  = %[2]q
  description = %[3]q
}
`, name, identifier, description)
}

func testAccProviderResourceConfig_oauth2Config(name, identifier string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name          = %[1]q
  zone_id       = keycard_zone.test.id
  identifier    = %[2]q
  client_id     = "test-client-id"
  client_secret = "test-client-secret"

  oauth2 = {
    authorization_endpoint = "%[2]s/authorize"
    token_endpoint         = "%[2]s/token"
  }
}
`, name, identifier)
}

func testAccProviderResourceConfig_withClientId(name, identifier, clientId string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[1]q
  zone_id    = keycard_zone.test.id
  identifier = %[2]q
  client_id  = %[3]q
}
`, name, identifier, clientId)
}

func testAccProviderResourceConfig_withClientSecret(name, identifier, clientSecret string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name          = %[1]q
  zone_id       = keycard_zone.test.id
  identifier    = %[2]q
  client_secret = %[3]q
}
`, name, identifier, clientSecret)
}

func testAccProviderResourceConfig_withOAuth2Endpoints(name, identifier, authEndpoint, tokenEndpoint string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[1]q
  zone_id    = keycard_zone.test.id
  identifier = %[2]q

  oauth2 = {
    authorization_endpoint = %[3]q
    token_endpoint         = %[4]q
  }
}
`, name, identifier, authEndpoint, tokenEndpoint)
}
