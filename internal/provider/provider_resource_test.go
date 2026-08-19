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
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing — identifier defaults from issuer
			{
				Config: testAccProviderResourceConfig_basic(rName, issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", issuer),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", issuer),
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
				Config: testAccProviderResourceConfig_basic(rName+"-updated", issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", issuer),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", issuer),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccProviderResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccProviderResourceConfig_withDescription(rName, issuer, "Test description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_provider.test", "description", "Test description"),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", issuer),
				),
			},
			// Update description
			{
				Config: testAccProviderResourceConfig_withDescription(rName, issuer, "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "description", "Updated description"),
				),
			},
			// Remove description
			{
				Config: testAccProviderResourceConfig_basic(rName, issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_provider.test", "description"),
				),
			},
		},
	})
}

func TestAccProviderResource_oauth2Config(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with full OAuth2 configuration
			{
				Config: testAccProviderResourceConfig_oauth2Config(rName, issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", issuer),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", issuer),
					resource.TestCheckResourceAttr("keycard_provider.test", "client_id", "test-client-id"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.authorization_endpoint", issuer+"/authorize"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.token_endpoint", issuer+"/token"),
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
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with basic config
			{
				Config: testAccProviderResourceConfig_basic(rName, issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", issuer),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", issuer),
				),
			},
			// Update to add client_id and oauth2 endpoints
			{
				Config: testAccProviderResourceConfig_oauth2Config(rName, issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", issuer),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", issuer),
					resource.TestCheckResourceAttr("keycard_provider.test", "client_id", "test-client-id"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.authorization_endpoint", issuer+"/authorize"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.token_endpoint", issuer+"/token"),
				),
			},
			// Update back to basic (remove optional fields)
			{
				Config: testAccProviderResourceConfig_basic(rName, issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", issuer),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", issuer),
					resource.TestCheckNoResourceAttr("keycard_provider.test", "client_id"),
					resource.TestCheckResourceAttrSet("keycard_provider.test", "oauth2.authorization_endpoint"),
					resource.TestCheckResourceAttrSet("keycard_provider.test", "oauth2.token_endpoint"),
				),
			},
		},
	})
}

func TestAccProviderResource_customIdentifier(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)
	identifier := rName + "-custom-id"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with explicit identifier different from issuer
			{
				Config: testAccProviderResourceConfig_withIdentifier(rName, issuer, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", issuer),
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

func TestAccProviderResource_identifierOnly(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)
	identifierV2 := fmt.Sprintf("https://%s-v2.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with identifier only (no oauth2 block) — backward compatible
			{
				Config: testAccProviderResourceConfig_identifierOnly(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
					// API copies identifier into protocols.oauth2.issuer
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", identifier),
					resource.TestCheckResourceAttrSet("keycard_provider.test", "id"),
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
			// Update name — identifier stays the same, no oauth2 in config
			{
				Config: testAccProviderResourceConfig_identifierOnly(rName+"-updated", identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifier),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", identifier),
				),
			},
			// Update identifier — oauth2.issuer must follow
			{
				Config: testAccProviderResourceConfig_identifierOnly(rName+"-updated", identifierV2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "identifier", identifierV2),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.issuer", identifierV2),
				),
			},
		},
	})
}

func TestAccProviderResource_missingIdentifierAndOAuth2Invalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_noIdentifierNoOAuth2(rName),
				ExpectError: regexp.MustCompile(`Missing Required Configuration`),
			},
		},
	})
}

func TestAccProviderResource_emptyDescriptionInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withDescription(rName, issuer, ""),
				ExpectError: regexp.MustCompile(`Attribute description string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_emptyClientIdInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withClientId(rName, issuer, ""),
				ExpectError: regexp.MustCompile(`Attribute client_id string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_emptyClientSecretInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withClientSecret(rName, issuer, ""),
				ExpectError: regexp.MustCompile(`Attribute client_secret string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_emptyAuthorizationEndpointInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withOAuth2Endpoints(rName, issuer, "", "https://token.example.com"),
				ExpectError: regexp.MustCompile(`Attribute oauth2.authorization_endpoint string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_emptyTokenEndpointInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_withOAuth2Endpoints(rName, issuer, "https://auth.example.com", ""),
				ExpectError: regexp.MustCompile(`Attribute oauth2.token_endpoint string length must be at least 1`),
			},
		},
	})
}

func TestAccProviderResource_invalidIssuerURI(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_basic(rName, "not-a-uri"),
				ExpectError: regexp.MustCompile(`Invalid URI`),
			},
		},
	})
}

func testAccProviderResourceConfig_basic(name, issuer string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer = %[2]q
  }
}
`, name, issuer)
}

func testAccProviderResourceConfig_withIdentifier(name, issuer, identifier string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q
  zone_id    = data.keycard_organization.test.zone_id
  identifier = %[3]q

  oauth2 = {
    issuer = %[2]q
  }
}
`, name, issuer, identifier)
}

func testAccProviderResourceConfig_withDescription(name, issuer, description string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name        = %[1]q
  zone_id     = data.keycard_organization.test.zone_id
  description = %[3]q

  oauth2 = {
    issuer = %[2]q
  }
}
`, name, issuer, description)
}

func testAccProviderResourceConfig_oauth2Config(name, issuer string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name          = %[1]q
  zone_id       = data.keycard_organization.test.zone_id
  client_id     = "test-client-id"
  client_secret = "test-client-secret"

  oauth2 = {
    issuer                 = %[2]q
    authorization_endpoint = "%[2]s/authorize"
    token_endpoint         = "%[2]s/token"
  }
}
`, name, issuer)
}

func testAccProviderResourceConfig_withClientId(name, issuer, clientId string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name      = %[1]q
  zone_id   = data.keycard_organization.test.zone_id
  client_id = %[3]q

  oauth2 = {
    issuer = %[2]q
  }
}
`, name, issuer, clientId)
}

func testAccProviderResourceConfig_withClientSecret(name, issuer, clientSecret string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name          = %[1]q
  zone_id       = data.keycard_organization.test.zone_id
  client_secret = %[3]q

  oauth2 = {
    issuer = %[2]q
  }
}
`, name, issuer, clientSecret)
}

func testAccProviderResourceConfig_withOAuth2Endpoints(name, issuer, authEndpoint, tokenEndpoint string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer                 = %[2]q
    authorization_endpoint = %[3]q
    token_endpoint         = %[4]q
  }
}
`, name, issuer, authEndpoint, tokenEndpoint)
}

func testAccProviderResourceConfig_identifierOnly(name, identifier string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q
  zone_id    = data.keycard_organization.test.zone_id
  identifier = %[2]q
}
`, name, identifier)
}

func testAccProviderResourceConfig_noIdentifierNoOAuth2(name string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = data.keycard_organization.test.zone_id
}
`, name)
}
