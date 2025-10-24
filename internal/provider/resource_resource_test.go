package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccResourceResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccResourceResourceConfig_basic(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "identifier", "https://"+rName+".example.com"),
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
				Config: testAccResourceResourceConfig_basic(zoneName, providerName, rName+"-updated"),
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
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccResourceResourceConfig_withDescription(zoneName, providerName, rName, "Test resource description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "description", "Test resource description"),
				),
			},
			// Update description
			{
				Config: testAccResourceResourceConfig_withDescription(zoneName, providerName, rName, "Updated resource description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "description", "Updated resource description"),
				),
			},
			// Remove description
			{
				Config: testAccResourceResourceConfig_basic(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_resource.test", "description"),
				),
			},
		},
	})
}

func TestAccResourceResource_withMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with metadata
			{
				Config: testAccResourceResourceConfig_withMetadata(zoneName, providerName, rName, "https://docs.example.com/resource"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "metadata.docs_url", "https://docs.example.com/resource"),
				),
			},
			// Update metadata docs_url
			{
				Config: testAccResourceResourceConfig_withMetadata(zoneName, providerName, rName, "https://docs.example.com/updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "metadata.docs_url", "https://docs.example.com/updated"),
				),
			},
			// Remove metadata
			{
				Config: testAccResourceResourceConfig_basic(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_resource.test", "metadata.docs_url"),
				),
			},
		},
	})
}

func TestAccResourceResource_withScopes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with OAuth2 scopes
			{
				Config: testAccResourceResourceConfig_withScopes(zoneName, providerName, rName, []string{"read"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.#", "1"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.0", "read"),
				),
			},
			// Update OAuth2 scopes (add more)
			{
				Config: testAccResourceResourceConfig_withScopes(zoneName, providerName, rName, []string{"read", "write"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.#", "2"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.0", "read"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.1", "write"),
				),
			},
			// Update OAuth2 scopes (change to single)
			{
				Config: testAccResourceResourceConfig_withScopes(zoneName, providerName, rName, []string{"admin"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.#", "1"),
					resource.TestCheckResourceAttr("keycard_resource.test", "oauth2.scopes.0", "admin"),
				),
			},
			// Remove OAuth2 block
			{
				Config: testAccResourceResourceConfig_basic(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_resource.test", "oauth2"),
				),
			},
		},
	})
}

func TestAccResourceResource_withApplication(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with application_id
			{
				Config: testAccResourceResourceConfig_withApplication(zoneName, providerName, appName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttrSet("keycard_resource.test", "application_id"),
					resource.TestCheckResourceAttrPair("keycard_resource.test", "application_id", "keycard_application.test", "id"),
				),
			},
			// Remove application_id
			{
				Config: testAccResourceResourceConfig_basic(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_resource.test", "application_id"),
				),
			},
		},
	})
}

func TestAccResourceResource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all fields
			{
				Config: testAccResourceResourceConfig_complete(zoneName, providerName, appName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_resource.test", "description", "Complete resource with all fields"),
					resource.TestCheckResourceAttr("keycard_resource.test", "identifier", "https://"+rName+".example.com"),
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

func TestAccResourceResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName1 := acctest.RandomWithPrefix("tftest-zone1")
	zoneName2 := acctest.RandomWithPrefix("tftest-zone2")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in zone 1
			{
				Config: testAccResourceResourceConfig_basic(zoneName1, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_resource.test", "zone_id", "keycard_zone.test", "id"),
				),
			},
			// Change zone (should force replacement)
			{
				Config: testAccResourceResourceConfig_basic(zoneName2, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_resource.test", "zone_id", "keycard_zone.test", "id"),
				),
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

func testAccResourceResourceConfig_basic(zoneName, providerName, resourceName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_resource" "test" {
  name                   = %[3]q
  identifier             = "https://%[3]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
}
`, zoneName, providerName, resourceName)
}

func testAccResourceResourceConfig_withDescription(zoneName, providerName, resourceName, description string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_resource" "test" {
  name                   = %[3]q
  description            = %[4]q
  identifier             = "https://%[3]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
}
`, zoneName, providerName, resourceName, description)
}

func testAccResourceResourceConfig_withMetadata(zoneName, providerName, resourceName, docsURL string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_resource" "test" {
  name                   = %[3]q
  identifier             = "https://%[3]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id

  metadata = {
    docs_url = %[4]q
  }
}
`, zoneName, providerName, resourceName, docsURL)
}

func testAccResourceResourceConfig_withScopes(zoneName, providerName, resourceName string, scopes []string) string {
	config := fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_resource" "test" {
  name                   = %[3]q
  identifier             = "https://%[3]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id

  oauth2 = {
    scopes = [
`, zoneName, providerName, resourceName)

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

func testAccResourceResourceConfig_withApplication(zoneName, providerName, appName, resourceName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_application" "test" {
  name       = %[3]q
  identifier = "https://%[3]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_resource" "test" {
  name                   = %[4]q
  identifier             = "https://%[4]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
  application_id         = keycard_application.test.id
}
`, zoneName, providerName, appName, resourceName)
}

func testAccResourceResourceConfig_complete(zoneName, providerName, appName, resourceName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_application" "test" {
  name       = %[3]q
  identifier = "https://%[3]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_resource" "test" {
  name                   = %[4]q
  description            = "Complete resource with all fields"
  identifier             = "https://%[4]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
  application_id         = keycard_application.test.id

  metadata = {
    docs_url = "https://docs.example.com/complete"
  }

  oauth2 = {
    scopes = ["read", "write"]
  }
}
`, zoneName, providerName, appName, resourceName)
}
