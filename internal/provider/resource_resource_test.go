package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccResourceResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccResourceResourceConfig_basic(providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "identifier", "https://"+rName+".example.com"),
					resource.TestCheckResourceAttr("keycard_resource.test", "prefix", "false"),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "credential_provider_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_resource.test",
				ImportState:       true,
				ImportStateIdFunc: testAccResourceImportStateIdFunc("keycard_resource.test"),
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccResourceResourceConfig_basic(providerName, rName+"-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("keycard_resource.test", "identifier", "https://"+rName+"-updated.example.com"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccResourceResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccResourceResourceConfig_withDescription(providerName, rName, "Test resource description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "description", "Test resource description"),
				),
			},
			// Update description
			{
				Config: testAccResourceResourceConfig_withDescription(providerName, rName, "Updated resource description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "description", "Updated resource description"),
				),
			},
			// Remove description
			{
				Config: testAccResourceResourceConfig_basic(providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_resource.test", "description"),
				),
			},
		},
	})
}

func TestAccResourceResource_withMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with metadata
			{
				Config: testAccResourceResourceConfig_withMetadata(providerName, rName, "https://docs.example.com/resource"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "metadata.docs_url", "https://docs.example.com/resource"),
				),
			},
			// Update metadata docs_url
			{
				Config: testAccResourceResourceConfig_withMetadata(providerName, rName, "https://docs.example.com/updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "metadata.docs_url", "https://docs.example.com/updated"),
				),
			},
			// Remove metadata
			{
				Config: testAccResourceResourceConfig_basic(providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_resource.test", "metadata.docs_url"),
				),
			},
		},
	})
}

func TestAccResourceResource_withScopes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with OAuth2 scopes
			{
				Config: testAccResourceResourceConfig_withScopes(providerName, rName, []string{"read"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.#", "1"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.0", "read"),
				),
			},
			// Update OAuth2 scopes (add more)
			{
				Config: testAccResourceResourceConfig_withScopes(providerName, rName, []string{"read", "write"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.#", "2"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.0", "read"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.1", "write"),
				),
			},
			// Update OAuth2 scopes (change to single)
			{
				Config: testAccResourceResourceConfig_withScopes(providerName, rName, []string{"admin"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.#", "1"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.0", "admin"),
				),
			},
			// Remove OAuth2 block
			{
				Config: testAccResourceResourceConfig_basic(providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_resource.test", "oauth2"),
				),
			},
		},
	})
}

func TestAccResourceResource_withApplication(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with application_id
			{
				Config: testAccResourceResourceConfig_withApplication(providerName, appName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "application_id"),
					resource.TestCheckResourceAttrPair("keycard_resource.test", "application_id", "keycard_application.test", "id"),
				),
			},
			// Remove application_id
			{
				Config: testAccResourceResourceConfig_basic(providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_resource.test", "application_id"),
				),
			},
		},
	})
}

func TestAccResourceResource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all fields
			{
				Config: testAccResourceResourceConfig_complete(providerName, appName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "description", "Complete resource with all fields"),
					resource.TestCheckResourceAttr("keycard_resource.test", "identifier", "https://"+rName+".example.com"),
					resource.TestCheckResourceAttr("keycard_resource.test", "prefix", "true"),
					resource.TestCheckResourceAttr("keycard_resource.test", "metadata.docs_url", "https://docs.example.com/complete"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.#", "2"),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "credential_provider_id"),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "application_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_resource.test",
				ImportState:       true,
				ImportStateIdFunc: testAccResourceImportStateIdFunc("keycard_resource.test"),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceResource_withPrefix(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with prefix = true
			{
				Config: testAccResourceResourceConfig_withPrefix(providerName, rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "prefix", "true"),
				),
			},
			// Toggle prefix to false
			{
				Config: testAccResourceResourceConfig_withPrefix(providerName, rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "prefix", "false"),
				),
			},
			// Remove prefix (should default to false)
			{
				Config: testAccResourceResourceConfig_basic(providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "prefix", "false"),
				),
			},
		},
	})
}

func TestAccResourceResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in the organization zone
			{
				Config: testAccResourceResourceConfig_inZone(providerName, rName, zoneName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_resource.test", "zone_id", testAccOrgZoneRef, "zone_id"),
				),
			},
			// Move to another zone (should force replacement)
			{
				Config: testAccResourceResourceConfig_inZone(providerName, rName, zoneName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_resource.test", "zone_id", testAccOtherZoneRef, "id"),
				),
			},
		},
	})
}

func TestAccResourceResource_emptyDescriptionInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceResourceConfig_withDescription(providerName, rName, ""),
				ExpectError: regexp.MustCompile(`Attribute description string length must be at least 1`),
			},
		},
	})
}

func TestAccResourceResource_emptyDocsUrlInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceResourceConfig_withMetadata(providerName, rName, ""),
				ExpectError: regexp.MustCompile(`Attribute metadata.docs_url string length must be at least 1`),
			},
		},
	})
}

func TestAccResourceResource_emptyApplicationIdInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceResourceConfig_withApplicationId(providerName, rName, ""),
				ExpectError: regexp.MustCompile(`Attribute application_id string length must be at least 1`),
			},
		},
	})
}

// Helper function to generate import state ID in format zones/{zone-id}/resources/{resource-id}.
func testAccResourceImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
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

		return fmt.Sprintf("zones/%s/resources/%s", zoneID, id), nil
	}
}

func testAccResourceResourceConfig_basic(providerName, resourceName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_resource" "test" {
  name                   = %[2]q
  identifier             = "https://%[2]s.example.com"
  zone_id                = data.keycard_organization.test.zone_id
  credential_provider_id = keycard_provider.test.id
}
`, providerName, resourceName)
}

func testAccResourceResourceConfig_withDescription(providerName, resourceName, description string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_resource" "test" {
  name                   = %[2]q
  description            = %[3]q
  identifier             = "https://%[2]s.example.com"
  zone_id                = data.keycard_organization.test.zone_id
  credential_provider_id = keycard_provider.test.id
}
`, providerName, resourceName, description)
}

func testAccResourceResourceConfig_withMetadata(providerName, resourceName, docsURL string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_resource" "test" {
  name                   = %[2]q
  identifier             = "https://%[2]s.example.com"
  zone_id                = data.keycard_organization.test.zone_id
  credential_provider_id = keycard_provider.test.id

  metadata = {
    docs_url = %[3]q
  }
}
`, providerName, resourceName, docsURL)
}

func testAccResourceResourceConfig_withScopes(providerName, resourceName string, scopes []string) string {
	config := testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_resource" "test" {
  name                   = %[2]q
  identifier             = "https://%[2]s.example.com"
  zone_id                = data.keycard_organization.test.zone_id
  credential_provider_id = keycard_provider.test.id

  oauth2 = {
    scopes = [
`, providerName, resourceName)

	for i, scope := range scopes {
		if i > 0 {
			config += ",\n"
		}
		config += fmt.Sprintf("      %q", scope)
	}

	config += `
    ]
  }
}
`
	return config
}

func testAccResourceResourceConfig_withApplication(providerName, appName, resourceName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_resource" "test" {
  name                   = %[3]q
  identifier             = "https://%[3]s.example.com"
  zone_id                = data.keycard_organization.test.zone_id
  credential_provider_id = keycard_provider.test.id
  application_id         = keycard_application.test.id
}
`, providerName, appName, resourceName)
}

func testAccResourceResourceConfig_complete(providerName, appName, resourceName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_resource" "test" {
  name                   = %[3]q
  description            = "Complete resource with all fields"
  identifier             = "https://%[3]s.example.com"
  zone_id                = data.keycard_organization.test.zone_id
  credential_provider_id = keycard_provider.test.id
  application_id         = keycard_application.test.id
  prefix                 = true

  metadata = {
    docs_url = "https://docs.example.com/complete"
  }

  oauth2 = {
    scopes = ["read", "write"]
  }
}
`, providerName, appName, resourceName)
}

func testAccResourceResourceConfig_withPrefix(providerName, resourceName string, prefix bool) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_resource" "test" {
  name                   = %[2]q
  identifier             = "https://%[2]s.example.com"
  zone_id                = data.keycard_organization.test.zone_id
  credential_provider_id = keycard_provider.test.id
  prefix                 = %[3]t
}
`, providerName, resourceName, prefix)
}

func testAccResourceResourceConfig_withApplicationId(providerName, resourceName, applicationId string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name       = %[1]q

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_resource" "test" {
  name                   = %[2]q
  identifier             = "https://%[2]s.example.com"
  zone_id                = data.keycard_organization.test.zone_id
  credential_provider_id = keycard_provider.test.id
  application_id         = %[3]q
}
`, providerName, resourceName, applicationId)
}

func testAccResourceResourceConfig_inZone(providerName, resourceName, zoneName string, otherZone bool) string {
	zoneConfig, zoneID := testAccZone(otherZone, zoneName)

	return zoneConfig + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = %[3]s

  oauth2 = {
    issuer = "https://%[1]s.example.com"
  }
}

resource "keycard_resource" "test" {
  name                   = %[2]q
  identifier             = "https://%[2]s.example.com"
  zone_id                = %[3]s
  credential_provider_id = keycard_provider.test.id
}
`, providerName, resourceName, zoneID)
}
