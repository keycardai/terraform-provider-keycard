package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationPublicCredentialResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationPublicCredentialResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "application_id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "client_id"),
					// Verify relationships
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "zone_id",
						"keycard_zone.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_application_public_credential.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["keycard_application_public_credential.test"]
					if !ok {
						return "", fmt.Errorf("Not found: keycard_application_public_credential.test")
					}
					zoneID := rs.Primary.Attributes["zone_id"]
					id := rs.Primary.ID
					return fmt.Sprintf("zones/%s/application-credentials/%s", zoneID, id), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccApplicationPublicCredentialResource_applicationChange(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tftest-app1")
	rName2 := acctest.RandomWithPrefix("tftest-app2")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first application
			{
				Config: testAccApplicationPublicCredentialResourceConfig_basic(zoneName, rName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
			// Change application (should force replacement)
			{
				Config: testAccApplicationPublicCredentialResourceConfig_basic(zoneName, rName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationPublicCredentialResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName1 := acctest.RandomWithPrefix("tftest-zone1")
	zoneName2 := acctest.RandomWithPrefix("tftest-zone2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in zone 1
			{
				Config: testAccApplicationPublicCredentialResourceConfig_basic(zoneName1, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "zone_id",
						"keycard_zone.test", "id",
					),
				),
			},
			// Change zone (should force replacement)
			{
				Config: testAccApplicationPublicCredentialResourceConfig_basic(zoneName2, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "zone_id",
						"keycard_zone.test", "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationPublicCredentialResource_multipleCredentials(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple credentials for the same application
			{
				Config: testAccApplicationPublicCredentialResourceConfig_multiple(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// First credential
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test1", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test1", "client_id"),
					// Second credential
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test2", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test2", "client_id"),
					// Both should be for the same application
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test1", "application_id",
						"keycard_application_public_credential.test2", "application_id",
					),
				),
			},
		},
	})
}

func testAccApplicationPublicCredentialResourceConfig_basic(zoneName, appName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_application_public_credential" "test" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
}
`, zoneName, appName)
}

func testAccApplicationPublicCredentialResourceConfig_multiple(zoneName, appName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_application_public_credential" "test1" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
}

resource "keycard_application_public_credential" "test2" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
}
`, zoneName, appName)
}
