package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneUserIdentityConfigDataSource_basic(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	identifier := fmt.Sprintf("https://%s.example.com", providerName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone with user identity config and fetch it with the data source
			{
				Config: testAccZoneUserIdentityConfigDataSourceConfig_basic(zoneName, providerName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match the resource
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone_user_identity_config.test", "zone_id",
						"keycard_zone_user_identity_config.test", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone_user_identity_config.test", "provider_id",
						"keycard_zone_user_identity_config.test", "provider_id",
					),
					// Verify data source attributes match the underlying resources
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone_user_identity_config.test", "zone_id",
						"keycard_zone.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone_user_identity_config.test", "provider_id",
						"keycard_provider.test", "id",
					),
					// Verify both attributes are set
					resource.TestCheckResourceAttrSet("data.keycard_zone_user_identity_config.test", "zone_id"),
					resource.TestCheckResourceAttrSet("data.keycard_zone_user_identity_config.test", "provider_id"),
				),
			},
		},
	})
}

func TestAccZoneUserIdentityConfigDataSource_noProviderConfigured(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch config for a zone without a configured provider
			{
				Config:      testAccZoneUserIdentityConfigDataSourceConfig_noProviderConfigured(zoneName),
				ExpectError: regexp.MustCompile("No Provider Configured"),
			},
		},
	})
}

func TestAccZoneUserIdentityConfigDataSource_zoneNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch config for a zone that doesn't exist
			{
				Config:      testAccZoneUserIdentityConfigDataSourceConfig_zoneNotFound(),
				ExpectError: regexp.MustCompile("Zone Not Found"),
			},
		},
	})
}

func testAccZoneUserIdentityConfigDataSourceConfig_basic(zoneName, providerName, identifier string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[2]q
  zone_id    = keycard_zone.test.id
  identifier = %[3]q
}

resource "keycard_zone_user_identity_config" "test" {
  zone_id     = keycard_zone.test.id
  provider_id = keycard_provider.test.id
}

data "keycard_zone_user_identity_config" "test" {
  zone_id = keycard_zone_user_identity_config.test.zone_id
}
`, zoneName, providerName, identifier)
}

func testAccZoneUserIdentityConfigDataSourceConfig_noProviderConfigured(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_zone_user_identity_config" "test" {
  zone_id = keycard_zone.test.id
}
`, zoneName)
}

func testAccZoneUserIdentityConfigDataSourceConfig_zoneNotFound() string {
	return `
data "keycard_zone_user_identity_config" "test" {
  zone_id = "non-existent-zone-id-12345"
}
`
}
