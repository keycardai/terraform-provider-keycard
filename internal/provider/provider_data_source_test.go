package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProviderDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a provider resource and fetch it with the data source
			{
				Config: testAccProviderDataSourceConfig_basic(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match the resource
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "id",
						"keycard_provider.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "zone_id",
						"keycard_provider.test", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "name",
						"keycard_provider.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "identifier",
						"keycard_provider.test", "identifier",
					),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "identifier", identifier),
				),
			},
		},
	})
}

func TestAccProviderDataSource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)
	description := "Test provider description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a provider with description and fetch it
			{
				Config: testAccProviderDataSourceConfig_withDescription(rName, identifier, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "id",
						"keycard_provider.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "description",
						"keycard_provider.test", "description",
					),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "description", description),
				),
			},
		},
	})
}

func TestAccProviderDataSource_withOAuth2(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a provider with OAuth2 configuration and fetch it
			{
				Config: testAccProviderDataSourceConfig_withOAuth2(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "id",
						"keycard_provider.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "client_id",
						"keycard_provider.test", "client_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "oauth2.authorization_endpoint",
						"keycard_provider.test", "oauth2.authorization_endpoint",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "oauth2.token_endpoint",
						"keycard_provider.test", "oauth2.token_endpoint",
					),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "client_id", "test-client-id"),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "oauth2.authorization_endpoint", identifier+"/authorize"),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "oauth2.token_endpoint", identifier+"/token"),
				),
			},
		},
	})
}

func TestAccProviderDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone but attempt to fetch a provider that doesn't exist
			{
				Config:      testAccProviderDataSourceConfig_notFound(rName),
				ExpectError: regexp.MustCompile("Provider Not Found"),
			},
		},
	})
}

func testAccProviderDataSourceConfig_basic(name, identifier string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[1]q
  zone_id    = keycard_zone.test.id
  identifier = %[2]q
}

data "keycard_provider" "test" {
  zone_id = keycard_provider.test.zone_id
  id      = keycard_provider.test.id
}
`, name, identifier)
}

func testAccProviderDataSourceConfig_withDescription(name, identifier, description string) string {
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

data "keycard_provider" "test" {
  zone_id = keycard_provider.test.zone_id
  id      = keycard_provider.test.id
}
`, name, identifier, description)
}

func testAccProviderDataSourceConfig_withOAuth2(name, identifier string) string {
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

data "keycard_provider" "test" {
  zone_id = keycard_provider.test.zone_id
  id      = keycard_provider.test.id
}
`, name, identifier)
}

func testAccProviderDataSourceConfig_notFound(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_provider" "test" {
  zone_id = keycard_zone.test.id
  id      = "non-existent-provider-id-12345"
}
`, name)
}
