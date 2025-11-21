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
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	urlValue := fmt.Sprintf("https://%s.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(zoneName, rName, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "application_id"),
					resource.TestCheckResourceAttr("keycard_application_url_credential.test", "url", urlValue),
					// Verify relationships
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "zone_id",
						"keycard_zone.test", "id",
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
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	urlValue1 := fmt.Sprintf("https://%s-1.example.com", rName)
	urlValue2 := fmt.Sprintf("https://%s-2.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first URL
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(zoneName, rName, urlValue1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttr("keycard_application_url_credential.test", "url", urlValue1),
				),
			},
			// Change URL (should force replacement)
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(zoneName, rName, urlValue2),
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
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	urlValue := "https://example.com/credential"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first application
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(zoneName, rName1, urlValue),
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
				Config: testAccApplicationURLCredentialResourceConfig_basic(zoneName, rName2, urlValue),
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
	zoneName1 := acctest.RandomWithPrefix("tftest-zone1")
	zoneName2 := acctest.RandomWithPrefix("tftest-zone2")
	urlValue := "https://example.com/credential"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in zone 1
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(zoneName1, rName, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "zone_id",
						"keycard_zone.test", "id",
					),
				),
			},
			// Change zone (should force replacement)
			{
				Config: testAccApplicationURLCredentialResourceConfig_basic(zoneName2, rName, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_url_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_url_credential.test", "zone_id",
						"keycard_zone.test", "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationURLCredentialResource_multipleCredentials(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	urlValue1 := fmt.Sprintf("https://%s-1.example.com", rName)
	urlValue2 := fmt.Sprintf("https://%s-2.example.com", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple credentials for the same application
			{
				Config: testAccApplicationURLCredentialResourceConfig_multiple(zoneName, rName, urlValue1, urlValue2),
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

func testAccApplicationURLCredentialResourceConfig_basic(zoneName, appName, urlValue string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_application_url_credential" "test" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  url            = %[3]q
}
`, zoneName, appName, urlValue)
}

func testAccApplicationURLCredentialResourceConfig_multiple(zoneName, appName, urlValue1, urlValue2 string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_application_url_credential" "test1" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  url            = %[3]q
}

resource "keycard_application_url_credential" "test2" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  url            = %[4]q
}
`, zoneName, appName, urlValue1, urlValue2)
}
