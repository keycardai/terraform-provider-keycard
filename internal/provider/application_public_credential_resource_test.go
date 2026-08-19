package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationPublicCredentialResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier := "client-" + rName

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationPublicCredentialResourceConfig_basic(rName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "application_id"),
					resource.TestCheckResourceAttr("keycard_application_public_credential.test", "identifier", identifier),
					// Verify relationships
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
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
	identifier := "client-" + acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first application
			{
				Config: testAccApplicationPublicCredentialResourceConfig_withApplication(rName1, rName2, identifier, "app1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "application_id",
						"keycard_application.app1", "id",
					),
				),
			},
			// Change application (should force replacement)
			{
				Config: testAccApplicationPublicCredentialResourceConfig_withApplication(rName1, rName2, identifier, "app2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "application_id",
						"keycard_application.app2", "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationPublicCredentialResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	identifier := "client-" + rName

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in the organization zone
			{
				Config: testAccApplicationPublicCredentialResourceConfig_inZone(rName, identifier, zoneName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
					),
				),
			},
			// Move to another zone (should force replacement)
			{
				Config: testAccApplicationPublicCredentialResourceConfig_inZone(rName, identifier, zoneName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_credential.test", "zone_id",
						testAccOtherZoneRef, "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationPublicCredentialResource_multipleCredentials(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	identifier1 := "client1-" + rName
	identifier2 := "client2-" + rName

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple credentials for the same application
			{
				Config: testAccApplicationPublicCredentialResourceConfig_multiple(rName, identifier1, identifier2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// First credential
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test1", "id"),
					resource.TestCheckResourceAttr("keycard_application_public_credential.test1", "identifier", identifier1),
					// Second credential
					resource.TestCheckResourceAttrSet("keycard_application_public_credential.test2", "id"),
					resource.TestCheckResourceAttr("keycard_application_public_credential.test2", "identifier", identifier2),
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

func TestAccApplicationPublicCredentialResource_emptyIdentifierInvalid(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApplicationPublicCredentialResourceConfig_withIdentifier(""),
				ExpectError: regexp.MustCompile(`Attribute identifier string length must be at least 1`),
			},
		},
	})
}

func testAccApplicationPublicCredentialResourceConfig_basic(appName, identifier string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_public_credential" "test" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  identifier     = %[2]q
}
`, appName, identifier)
}

func testAccApplicationPublicCredentialResourceConfig_multiple(appName, identifier1, identifier2 string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_public_credential" "test1" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  identifier     = %[2]q
}

resource "keycard_application_public_credential" "test2" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  identifier     = %[3]q
}
`, appName, identifier1, identifier2)
}

func testAccApplicationPublicCredentialResourceConfig_withIdentifier(identifier string) string {
	return fmt.Sprintf(`
resource "keycard_application_public_credential" "test" {
  zone_id        = "stub-zone-id"
  application_id = "stub-app-id"
  identifier     = %q
}
`, identifier)
}

func testAccApplicationPublicCredentialResourceConfig_withApplication(appName1, appName2, identifier, appResourceName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "app1" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application" "app2" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_public_credential" "test" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.%[3]s.id
  identifier     = %[4]q
}
`, appName1, appName2, appResourceName, identifier)
}

func testAccApplicationPublicCredentialResourceConfig_inZone(appName, identifier, zoneName string, otherZone bool) string {
	zoneConfig, zoneID := testAccZone(otherZone, zoneName)

	return zoneConfig + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = %[3]s
}

resource "keycard_application_public_credential" "test" {
  zone_id        = %[3]s
  application_id = keycard_application.test.id
  identifier     = %[2]q
}
`, appName, identifier, zoneID)
}
