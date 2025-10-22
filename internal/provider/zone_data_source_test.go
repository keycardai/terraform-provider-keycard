package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone resource and fetch it with the data source
			{
				Config: testAccZoneDataSourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match the resource
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "id",
						"keycard_zone.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "name",
						"keycard_zone.test", "name",
					),
					resource.TestCheckResourceAttr("data.keycard_zone.test", "name", rName),
					// Verify OAuth2 values are fetched
					resource.TestCheckResourceAttrSet("data.keycard_zone.test", "oauth2.pkce_required"),
					resource.TestCheckResourceAttrSet("data.keycard_zone.test", "oauth2.dcr_enabled"),
					// Verify OAuth2 protocol URIs are fetched
					resource.TestCheckResourceAttrSet("data.keycard_zone.test", "oauth2.issuer_uri"),
					resource.TestCheckResourceAttrSet("data.keycard_zone.test", "oauth2.redirect_uri"),
				),
			},
		},
	})
}

func TestAccZoneDataSource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	description := "Test zone description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone with description and fetch it
			{
				Config: testAccZoneDataSourceConfig_withDescription(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "id",
						"keycard_zone.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "description",
						"keycard_zone.test", "description",
					),
					resource.TestCheckResourceAttr("data.keycard_zone.test", "description", description),
				),
			},
		},
	})
}

func TestAccZoneDataSource_oauth2(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone with custom OAuth2 settings and fetch it
			{
				Config: testAccZoneDataSourceConfig_withOAuth2(rName, false, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "id",
						"keycard_zone.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "oauth2.pkce_required",
						"keycard_zone.test", "oauth2.pkce_required",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "oauth2.dcr_enabled",
						"keycard_zone.test", "oauth2.dcr_enabled",
					),
					resource.TestCheckResourceAttr("data.keycard_zone.test", "oauth2.pkce_required", "false"),
					resource.TestCheckResourceAttr("data.keycard_zone.test", "oauth2.dcr_enabled", "true"),
					// Verify OAuth2 protocol URIs match between resource and data source
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "oauth2.issuer_uri",
						"keycard_zone.test", "oauth2.issuer_uri",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone.test", "oauth2.redirect_uri",
						"keycard_zone.test", "oauth2.redirect_uri",
					),
				),
			},
		},
	})
}

func TestAccZoneDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch a zone that doesn't exist
			{
				Config:      testAccZoneDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile("Zone Not Found"),
			},
		},
	})
}

func testAccZoneDataSourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_zone" "test" {
  id = keycard_zone.test.id
}
`, name)
}

func testAccZoneDataSourceConfig_withDescription(name, description string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name        = %[1]q
  description = %[2]q
}

data "keycard_zone" "test" {
  id = keycard_zone.test.id
}
`, name, description)
}

func testAccZoneDataSourceConfig_withOAuth2(name string, pkceRequired, dcrEnabled bool) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q

  oauth2 = {
    pkce_required = %[2]t
    dcr_enabled   = %[3]t
  }
}

data "keycard_zone" "test" {
  id = keycard_zone.test.id
}
`, name, pkceRequired, dcrEnabled)
}

func testAccZoneDataSourceConfig_notFound() string {
	return `
data "keycard_zone" "test" {
  id = "non-existent-zone-id-12345"
}
`
}
