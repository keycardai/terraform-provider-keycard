package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "identifier", "https://"+rName+".example.com"),
					resource.TestCheckResourceAttrSet("keycard_application.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application.test", "zone_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_application.test",
				ImportState:       true,
				ImportStateIdFunc: testAccApplicationImportStateIdFunc("keycard_application.test"),
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccApplicationResourceConfig_basic(zoneName, rName+"-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("keycard_application.test", "identifier", "https://"+rName+"-updated.example.com"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccApplicationResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccApplicationResourceConfig_withDescription(zoneName, rName, "Test application description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "description", "Test application description"),
				),
			},
			// Update description
			{
				Config: testAccApplicationResourceConfig_withDescription(zoneName, rName, "Updated application description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "description", "Updated application description"),
				),
			},
			// Remove description
			{
				Config: testAccApplicationResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_application.test", "description"),
				),
			},
		},
	})
}

func TestAccApplicationResource_withMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with metadata
			{
				Config: testAccApplicationResourceConfig_withMetadata(zoneName, rName, "https://docs.example.com/app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "metadata.docs_url", "https://docs.example.com/app"),
				),
			},
			// Update metadata docs_url
			{
				Config: testAccApplicationResourceConfig_withMetadata(zoneName, rName, "https://docs.example.com/updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "metadata.docs_url", "https://docs.example.com/updated"),
				),
			},
			// Remove metadata
			{
				Config: testAccApplicationResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_application.test", "metadata.docs_url"),
				),
			},
		},
	})
}

func TestAccApplicationResource_withOAuth2(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with OAuth2 redirect URIs
			{
				Config: testAccApplicationResourceConfig_withOAuth2(zoneName, rName, []string{
					"https://" + rName + ".example.com/callback",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.#", "1"),
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.0", "https://"+rName+".example.com/callback"),
				),
			},
			// Update OAuth2 redirect URIs (add more)
			{
				Config: testAccApplicationResourceConfig_withOAuth2(zoneName, rName, []string{
					"https://" + rName + ".example.com/callback",
					"https://" + rName + ".example.com/auth/callback",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.#", "2"),
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.0", "https://"+rName+".example.com/callback"),
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.1", "https://"+rName+".example.com/auth/callback"),
				),
			},
			// Update OAuth2 redirect URIs (change to single)
			{
				Config: testAccApplicationResourceConfig_withOAuth2(zoneName, rName, []string{
					"https://" + rName + ".example.com/new-callback",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.#", "1"),
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.0", "https://"+rName+".example.com/new-callback"),
				),
			},
			// Remove OAuth2 block
			{
				Config: testAccApplicationResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_application.test", "oauth2"),
				),
			},
		},
	})
}

func TestAccApplicationResource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all fields
			{
				Config: testAccApplicationResourceConfig_complete(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "description", "Complete application with all fields"),
					resource.TestCheckResourceAttr("keycard_application.test", "identifier", "https://"+rName+".example.com"),
					resource.TestCheckResourceAttr("keycard_application.test", "metadata.docs_url", "https://docs.example.com/complete"),
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.#", "2"),
					resource.TestCheckResourceAttrSet("keycard_application.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application.test", "zone_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_application.test",
				ImportState:       true,
				ImportStateIdFunc: testAccApplicationImportStateIdFunc("keycard_application.test"),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccApplicationResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName1 := acctest.RandomWithPrefix("tftest-zone1")
	zoneName2 := acctest.RandomWithPrefix("tftest-zone2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in zone 1
			{
				Config: testAccApplicationResourceConfig_basic(zoneName1, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_application.test", "zone_id", "keycard_zone.test", "id"),
				),
			},
			// Change zone (should force replacement)
			{
				Config: testAccApplicationResourceConfig_basic(zoneName2, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_application.test", "zone_id", "keycard_zone.test", "id"),
				),
			},
		},
	})
}

// Helper function to generate import state ID in format zones/{zone-id}/applications/{application-id}.
func testAccApplicationImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		id := rs.Primary.ID

		if zoneID == "" || id == "" {
			return "", fmt.Errorf("zone_id or id is empty")
		}

		return fmt.Sprintf("zones/%s/applications/%s", zoneID, id), nil
	}
}

func testAccApplicationResourceConfig_basic(zoneName, appName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}
`, zoneName, appName)
}

func testAccApplicationResourceConfig_withDescription(zoneName, appName, description string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name        = %[2]q
  description = %[3]q
  identifier  = "https://%[2]s.example.com"
  zone_id     = keycard_zone.test.id
}
`, zoneName, appName, description)
}

func testAccApplicationResourceConfig_withMetadata(zoneName, appName, docsURL string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id

  metadata = {
    docs_url = %[3]q
  }
}
`, zoneName, appName, docsURL)
}

func testAccApplicationResourceConfig_withOAuth2(zoneName, appName string, redirectURIs []string) string {
	config := fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id

  oauth2 = {
    redirect_uris = [
`, zoneName, appName)

	for i, uri := range redirectURIs {
		if i > 0 {
			config += ",\n"
		}
		config += fmt.Sprintf("      %q", uri)
	}

	config += `
    ]
  }
}
`
	return config
}

func testAccApplicationResourceConfig_complete(zoneName, appName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name        = %[2]q
  description = "Complete application with all fields"
  identifier  = "https://%[2]s.example.com"
  zone_id     = keycard_zone.test.id

  metadata = {
    docs_url = "https://docs.example.com/complete"
  }

  oauth2 = {
    redirect_uris = [
      "https://%[2]s.example.com/callback",
      "https://%[2]s.example.com/auth/callback"
    ]
  }
}
`, zoneName, appName)
}
