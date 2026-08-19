package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationClientSecretResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationClientSecretResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "application_id"),
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "client_id"),
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "client_secret"),
					// Verify relationships
					resource.TestCheckResourceAttrPair(
						"keycard_application_client_secret.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_application_client_secret.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccApplicationClientSecretResource_applicationChange(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tftest-app1")
	rName2 := acctest.RandomWithPrefix("tftest-app2")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first application
			{
				Config: testAccApplicationClientSecretResourceConfig_basic(rName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_client_secret.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
			// Change application (should force replacement)
			{
				Config: testAccApplicationClientSecretResourceConfig_basic(rName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_client_secret.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationClientSecretResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in the organization zone
			{
				Config: testAccApplicationClientSecretResourceConfig_inZone(rName, zoneName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_client_secret.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
					),
				),
			},
			// Move to another zone (should force replacement)
			{
				Config: testAccApplicationClientSecretResourceConfig_inZone(rName, zoneName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_client_secret.test", "zone_id",
						testAccOtherZoneRef, "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationClientSecretResource_multipleCredentials(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple credentials for the same application
			{
				Config: testAccApplicationClientSecretResourceConfig_multiple(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// First credential
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test1", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test1", "client_id"),
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test1", "client_secret"),
					// Second credential
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test2", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test2", "client_id"),
					resource.TestCheckResourceAttrSet("keycard_application_client_secret.test2", "client_secret"),
					// Both should be for the same application
					resource.TestCheckResourceAttrPair(
						"keycard_application_client_secret.test1", "application_id",
						"keycard_application_client_secret.test2", "application_id",
					),
				),
			},
		},
	})
}

func testAccApplicationClientSecretResourceConfig_basic(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_client_secret" "test" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
}
`, appName)
}

func testAccApplicationClientSecretResourceConfig_multiple(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_client_secret" "test1" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
}

resource "keycard_application_client_secret" "test2" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
}
`, appName)
}

func testAccApplicationClientSecretResourceConfig_inZone(appName, zoneName string, otherZone bool) string {
	zoneConfig, zoneID := testAccZone(otherZone, zoneName)

	return zoneConfig + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = %[2]s
}

resource "keycard_application_client_secret" "test" {
  zone_id        = %[2]s
  application_id = keycard_application.test.id
}
`, appName, zoneID)
}
