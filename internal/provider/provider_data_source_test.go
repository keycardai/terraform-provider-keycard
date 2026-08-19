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
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a provider resource and fetch it with the data source
			{
				Config: testAccProviderDataSourceConfig_basic(rName, issuer),
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
					resource.TestCheckResourceAttrPair(
						"data.keycard_provider.test", "oauth2.issuer",
						"keycard_provider.test", "oauth2.issuer",
					),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "identifier", issuer),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "oauth2.issuer", issuer),
				),
			},
		},
	})
}

func TestAccProviderDataSource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)
	description := "Test provider description"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a provider with description and fetch it
			{
				Config: testAccProviderDataSourceConfig_withDescription(rName, issuer, description),
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
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a provider with OAuth2 configuration and fetch it
			{
				Config: testAccProviderDataSourceConfig_withOAuth2(rName, issuer),
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
						"data.keycard_provider.test", "oauth2.issuer",
						"keycard_provider.test", "oauth2.issuer",
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
					resource.TestCheckResourceAttr("data.keycard_provider.test", "oauth2.issuer", issuer),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "oauth2.authorization_endpoint", issuer+"/authorize"),
					resource.TestCheckResourceAttr("data.keycard_provider.test", "oauth2.token_endpoint", issuer+"/token"),
				),
			},
		},
	})
}

func TestAccProviderDataSource_notFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch a provider that doesn't exist
			{
				Config:      testAccProviderDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile("Provider Not Found"),
			},
		},
	})
}

func testAccProviderDataSourceConfig_basic(name, issuer string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer = %[2]q
  }
}

data "keycard_provider" "test" {
  zone_id = keycard_provider.test.zone_id
  id      = keycard_provider.test.id
}
`, name, issuer)
}

func testAccProviderDataSourceConfig_withDescription(name, issuer, description string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name        = %[1]q
  zone_id     = data.keycard_organization.test.zone_id
  description = %[3]q

  oauth2 = {
    issuer = %[2]q
  }
}

data "keycard_provider" "test" {
  zone_id = keycard_provider.test.zone_id
  id      = keycard_provider.test.id
}
`, name, issuer, description)
}

func testAccProviderDataSourceConfig_withOAuth2(name, issuer string) string {
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

data "keycard_provider" "test" {
  zone_id = keycard_provider.test.zone_id
  id      = keycard_provider.test.id
}
`, name, issuer)
}

func testAccProviderDataSourceConfig_notFound() string {
	return testAccOrgZone + `
data "keycard_provider" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "non-existent-provider-id-12345"
}
`
}
