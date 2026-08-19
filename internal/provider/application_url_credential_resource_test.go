package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationURLCredentialResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	urlValue := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(rName, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "application_id"),
					resource.TestCheckResourceAttr("keycard_application_url_credential.test", "url", urlValue),
					// Verify relationships
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_application_url_credential.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["keycard_application_url_credential.test"]
					if !ok {
						return "", fmt.Errorf("Not found: keycard_application_url_credential.test")
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

func TestAccApplicationURLCredentialResource_urlChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	urlValue1 := fmt.Sprintf("https://%s-1.example.com", rName)
	urlValue2 := fmt.Sprintf("https://%s-2.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first URL
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(rName, urlValue1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttr("keycard_application_url_credential.test", "url", urlValue1),
				),
			},
			// Change URL (should force replacement)
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(rName, urlValue2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttr("keycard_application_url_credential.test", "url", urlValue2),
				),
			},
		},
	})
}

func TestAccApplicationURLCredentialResource_applicationChange(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tftest-app1")
	rName2 := acctest.RandomWithPrefix("tftest-app2")
	urlValue := "https://example.com/credential"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first application
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(rName1, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
			// Change application (should force replacement)
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(rName2, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationURLCredentialResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in the organization zone
			{
				Config: testAccApplicationURLCredentialResourceConfig_inZone(rName, zoneName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
					),
				),
			},
			// Move to another zone (should force replacement)
			{
				Config: testAccApplicationURLCredentialResourceConfig_inZone(rName, zoneName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "zone_id",
						testAccOtherZoneRef, "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationURLCredentialResource_multipleCredentials(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	urlValue1 := fmt.Sprintf("https://%s-1.example.com", rName)
	urlValue2 := fmt.Sprintf("https://%s-2.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple credentials for the same application
			{
				Config: testAccApplicationURLCredentialResourceConfig_multiple(rName, urlValue1, urlValue2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// First credential
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test1", "id"),
					resource.TestCheckResourceAttr("keycard_application_url_credential.test1", "url", urlValue1),
					// Second credential
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test2", "id"),
					resource.TestCheckResourceAttr("keycard_application_url_credential.test2", "url", urlValue2),
					// Both should be for the same application
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test1", "application_id",
						"keycard_application_url_credential.test2", "application_id",
					),
				),
			},
		},
	})
}

func testAccApplicationURLCredentialResourceConfig_basic(appName, urlValue string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_url_credential" "test" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  url            = %[2]q
}
`, appName, urlValue)
}

func testAccApplicationURLCredentialResourceConfig_multiple(appName, urlValue1, urlValue2 string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_url_credential" "test1" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  url            = %[2]q
}

resource "keycard_application_url_credential" "test2" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  url            = %[3]q
}
`, appName, urlValue1, urlValue2)
}

func testAccApplicationURLCredentialResourceConfig_inZone(appName, zoneName string, otherZone bool) string {
	zoneConfig, zoneID := testAccZone(otherZone, zoneName)

	return zoneConfig + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = %[2]s
}

resource "keycard_application_url_credential" "test" {
  zone_id        = %[2]s
  application_id = keycard_application.test.id
  url            = "https://example.com/credential"
}
`, appName, zoneID)
}
