package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccZoneUserIdentityConfigResource_basic(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	identifier := fmt.Sprintf("https://%s.example.com", providerName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccZoneUserIdentityConfigResourceConfig_basic(zoneName, providerName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_zone_user_identity_config.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_zone_user_identity_config.test", "provider_id"),
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "zone_id",
						"keycard_zone.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "provider_id",
						"keycard_provider.test", "id",
					),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "keycard_zone_user_identity_config.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    testAccZoneUserIdentityConfigImportStateIdFunc,
				ImportStateVerifyIdentifierAttribute: "zone_id",
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccZoneUserIdentityConfigResource_updateProvider(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest")
	provider1Name := acctest.RandomWithPrefix("tftest-provider1")
	provider2Name := acctest.RandomWithPrefix("tftest-provider2")
	identifier1 := fmt.Sprintf("https://%s.example.com", provider1Name)
	identifier2 := fmt.Sprintf("https://%s.example.com", provider2Name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first provider
			{
				Config: testAccZoneUserIdentityConfigResourceConfig_basic(zoneName, provider1Name, identifier1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "zone_id",
						"keycard_zone.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "provider_id",
						"keycard_provider.test", "id",
					),
				),
			},
			// Update to use second provider
			{
				Config: testAccZoneUserIdentityConfigResourceConfig_twoProviders(zoneName, provider1Name, identifier1, provider2Name, identifier2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "zone_id",
						"keycard_zone.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "provider_id",
						"keycard_provider.test2", "id",
					),
				),
			},
		},
	})
}

func TestAccZoneUserIdentityConfigResource_replaceOnZoneChange(t *testing.T) {
	zone1Name := acctest.RandomWithPrefix("tftest-zone1")
	zone2Name := acctest.RandomWithPrefix("tftest-zone2")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	identifier := fmt.Sprintf("https://%s.example.com", providerName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first zone
			{
				Config: testAccZoneUserIdentityConfigResourceConfig_twoZones(zone1Name, zone2Name, providerName, identifier, "test1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "zone_id",
						"keycard_zone.test1", "id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "provider_id",
						"keycard_provider.test", "id",
					),
				),
			},
			// Change to second zone - should force replacement
			{
				Config: testAccZoneUserIdentityConfigResourceConfig_twoZones(zone1Name, zone2Name, providerName, identifier, "test2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "zone_id",
						"keycard_zone.test2", "id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_zone_user_identity_config.test", "provider_id",
						"keycard_provider.test", "id",
					),
				),
			},
		},
	})
}

func testAccZoneUserIdentityConfigImportStateIdFunc(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["keycard_zone_user_identity_config.test"]
	if !ok {
		return "", fmt.Errorf("resource not found: keycard_zone_user_identity_config.test")
	}

	// Import ID is the zone_id
	return rs.Primary.Attributes["zone_id"], nil
}

func testAccZoneUserIdentityConfigResourceConfig_basic(zoneName, providerName, identifier string) string {
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
`, zoneName, providerName, identifier)
}

func testAccZoneUserIdentityConfigResourceConfig_twoProviders(zoneName, provider1Name, identifier1, provider2Name, identifier2 string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[2]q
  zone_id    = keycard_zone.test.id
  identifier = %[3]q
}

resource "keycard_provider" "test2" {
  name       = %[4]q
  zone_id    = keycard_zone.test.id
  identifier = %[5]q
}

resource "keycard_zone_user_identity_config" "test" {
  zone_id     = keycard_zone.test.id
  provider_id = keycard_provider.test2.id
}
`, zoneName, provider1Name, identifier1, provider2Name, identifier2)
}

func testAccZoneUserIdentityConfigResourceConfig_twoZones(zone1Name, zone2Name, providerName, identifier, identityZoneId string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test1" {
  name = %[1]q
}

resource "keycard_zone" "test2" {
  name = %[2]q
}

resource "keycard_provider" "test" {
  name       = %[3]q
  zone_id    = keycard_zone.%[5]s.id
  identifier = %[4]q
}

resource "keycard_zone_user_identity_config" "test" {
  zone_id     = keycard_zone.%[5]s.id
  provider_id = keycard_provider.test.id
}
`, zone1Name, zone2Name, providerName, identifier, identityZoneId)
}
