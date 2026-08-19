package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "identifier", "https://"+rName+".example.com"),
					resource.TestCheckResourceAttr("keycard_application.test", "consent", "required"),
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
				Config: testAccApplicationResourceConfig_basic(rName + "-updated"),
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

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccApplicationResourceConfig_withDescription(rName, "Test application description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "description", "Test application description"),
				),
			},
			// Update description
			{
				Config: testAccApplicationResourceConfig_withDescription(rName, "Updated application description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "description", "Updated application description"),
				),
			},
			// Remove description
			{
				Config: testAccApplicationResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_application.test", "description"),
				),
			},
		},
	})
}

func TestAccApplicationResource_withMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with metadata
			{
				Config: testAccApplicationResourceConfig_withMetadata(rName, "https://docs.example.com/app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "metadata.docs_url", "https://docs.example.com/app"),
				),
			},
			// Update metadata docs_url
			{
				Config: testAccApplicationResourceConfig_withMetadata(rName, "https://docs.example.com/updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "metadata.docs_url", "https://docs.example.com/updated"),
				),
			},
			// Remove metadata
			{
				Config: testAccApplicationResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_application.test", "metadata.docs_url"),
				),
			},
		},
	})
}

func TestAccApplicationResource_withOAuth2(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with OAuth2 redirect URIs
			{
				Config: testAccApplicationResourceConfig_withOAuth2(rName, []string{
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
				Config: testAccApplicationResourceConfig_withOAuth2(rName, []string{
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
				Config: testAccApplicationResourceConfig_withOAuth2(rName, []string{
					"https://" + rName + ".example.com/new-callback",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.#", "1"),
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.0", "https://"+rName+".example.com/new-callback"),
				),
			},
			// Remove OAuth2 block
			{
				Config: testAccApplicationResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_application.test", "oauth2"),
				),
			},
		},
	})
}

func TestAccApplicationResource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all fields
			{
				Config: testAccApplicationResourceConfig_complete(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "consent", "implicit"),
					resource.TestCheckResourceAttr("keycard_application.test", "description", "Complete application with all fields"),
					resource.TestCheckResourceAttr("keycard_application.test", "identifier", "https://"+rName+".example.com"),
					resource.TestCheckResourceAttr("keycard_application.test", "metadata.docs_url", "https://docs.example.com/complete"),
					resource.TestCheckResourceAttr("keycard_application.test", "oauth2.redirect_uris.#", "2"),
					resource.TestCheckResourceAttr("keycard_application.test", "traits.#", "1"),
					resource.TestCheckResourceAttr("keycard_application.test", "traits.0", "gateway"),
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
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in the organization zone
			{
				Config: testAccApplicationResourceConfig_inZone(rName, zoneName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_application.test", "zone_id", testAccOrgZoneRef, "zone_id"),
				),
			},
			// Move to another zone (should force replacement)
			{
				Config: testAccApplicationResourceConfig_inZone(rName, zoneName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_application.test", "zone_id", testAccOtherZoneRef, "id"),
				),
			},
		},
	})
}

func TestAccApplicationResource_withConsent(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with consent = "implicit"
			{
				Config: testAccApplicationResourceConfig_withConsent(rName, "implicit"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "consent", "implicit"),
				),
			},
			// Update consent to "required"
			{
				Config: testAccApplicationResourceConfig_withConsent(rName, "required"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "consent", "required"),
				),
			},
			// Remove consent (should default to "required")
			{
				Config: testAccApplicationResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "consent", "required"),
				),
			},
		},
	})
}

func TestAccApplicationResource_invalidConsentValue(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApplicationResourceConfig_withConsent(rName, "invalid"),
				ExpectError: regexp.MustCompile(`Attribute consent value must be one of`),
			},
		},
	})
}

func TestAccApplicationResource_emptyConsentInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApplicationResourceConfig_withConsent(rName, ""),
				ExpectError: regexp.MustCompile(`Attribute consent string length must be at least 1`),
			},
		},
	})
}

func TestAccApplicationResource_emptyDescriptionInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApplicationResourceConfig_withDescription(rName, ""),
				ExpectError: regexp.MustCompile(`Attribute description string length must be at least 1`),
			},
		},
	})
}

func TestAccApplicationResource_emptyDocsUrlInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApplicationResourceConfig_withMetadata(rName, ""),
				ExpectError: regexp.MustCompile(`Attribute metadata.docs_url string length must be at least 1`),
			},
		},
	})
}

func TestAccApplicationResource_withTraits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with traits
			{
				Config: testAccApplicationResourceConfig_withTraits(rName, []string{"gateway"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_application.test", "traits.#", "1"),
					resource.TestCheckResourceAttr("keycard_application.test", "traits.0", "gateway"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_application.test",
				ImportState:       true,
				ImportStateIdFunc: testAccApplicationImportStateIdFunc("keycard_application.test"),
				ImportStateVerify: true,
			},
			// Remove traits
			{
				Config: testAccApplicationResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_application.test", "traits"),
				),
			},
			// Re-add traits
			{
				Config: testAccApplicationResourceConfig_withTraits(rName, []string{"gateway"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_application.test", "traits.#", "1"),
					resource.TestCheckResourceAttr("keycard_application.test", "traits.0", "gateway"),
				),
			},
		},
	})
}

func TestAccApplicationResource_invalidTraitValue(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApplicationResourceConfig_withTraits(rName, []string{"invalid-trait"}),
				ExpectError: regexp.MustCompile(`Attribute traits\[0\] value must be one of`),
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

func testAccApplicationResourceConfig_basic(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}
`, appName)
}

func testAccApplicationResourceConfig_inZone(appName, zoneName string, otherZone bool) string {
	zoneConfig, zoneID := testAccZone(otherZone, zoneName)

	return zoneConfig + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = %[2]s
}
`, appName, zoneID)
}

func testAccApplicationResourceConfig_withDescription(appName, description string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name        = %[1]q
  description = %[2]q
  identifier  = "https://%[1]s.example.com"
  zone_id     = data.keycard_organization.test.zone_id
}
`, appName, description)
}

func testAccApplicationResourceConfig_withMetadata(appName, docsURL string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id

  metadata = {
    docs_url = %[2]q
  }
}
`, appName, docsURL)
}

func testAccApplicationResourceConfig_withOAuth2(appName string, redirectURIs []string) string {
	config := testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id

  oauth2 = {
    redirect_uris = [
`, appName)

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

func testAccApplicationResourceConfig_withTraits(appName string, traits []string) string {
	config := testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id

  traits = [
`, appName)

	for i, trait := range traits {
		if i > 0 {
			config += ",\n"
		}
		config += fmt.Sprintf("    %q", trait)
	}

	config += `
  ]
}
`
	return config
}

func testAccApplicationResourceConfig_withConsent(appName, consent string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  consent    = %[2]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}
`, appName, consent)
}

func testAccApplicationResourceConfig_complete(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name        = %[1]q
  consent     = "implicit"
  description = "Complete application with all fields"
  identifier  = "https://%[1]s.example.com"
  zone_id     = data.keycard_organization.test.zone_id

  metadata = {
    docs_url = "https://docs.example.com/complete"
  }

  oauth2 = {
    redirect_uris = [
      "https://%[1]s.example.com/callback",
      "https://%[1]s.example.com/auth/callback"
    ]
  }

  traits = ["gateway"]
}
`, appName)
}
