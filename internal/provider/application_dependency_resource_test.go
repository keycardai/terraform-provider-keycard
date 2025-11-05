package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationDependencyResource_basic(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")
	resourceName := acctest.RandomWithPrefix("tftest-resource")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationDependencyResourceConfig_basic(zoneName, providerName, appName, resourceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "zone_id", "keycard_zone.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "application_id", "keycard_application.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "resource_id", "keycard_resource.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "keycard_application_dependency.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccApplicationDependencyImportStateIdFunc("keycard_application_dependency.test"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "resource_id",
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccApplicationDependencyResource_multipleResources(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")
	resourceName1 := acctest.RandomWithPrefix("tftest-resource1")
	resourceName2 := acctest.RandomWithPrefix("tftest-resource2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create dependencies for multiple resources
			{
				Config: testAccApplicationDependencyResourceConfig_multiple(zoneName, providerName, appName, resourceName1, resourceName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test1", "application_id", "keycard_application.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test2", "application_id", "keycard_application.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test1", "resource_id", "keycard_resource.test1", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test2", "resource_id", "keycard_resource.test2", "id"),
				),
			},
		},
	})
}

func TestAccApplicationDependencyResource_resourceChange(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")
	resourceName1 := acctest.RandomWithPrefix("tftest-resource1")
	resourceName2 := acctest.RandomWithPrefix("tftest-resource2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create dependency with first resource
			{
				Config: testAccApplicationDependencyResourceConfig_withTwoResources(zoneName, providerName, appName, resourceName1, resourceName2, "test1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "resource_id", "keycard_resource.test1", "id"),
				),
			},
			// Change to second resource (should force replacement)
			{
				Config: testAccApplicationDependencyResourceConfig_withTwoResources(zoneName, providerName, appName, resourceName1, resourceName2, "test2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "resource_id", "keycard_resource.test2", "id"),
				),
			},
		},
	})
}

func TestAccApplicationDependencyResource_whenAccessing(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")
	resourceName := acctest.RandomWithPrefix("tftest-resource")
	additionalResourceName1 := acctest.RandomWithPrefix("tftest-additional1")
	additionalResourceName2 := acctest.RandomWithPrefix("tftest-additional2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create dependency with when_accessing
			{
				Config: testAccApplicationDependencyResourceConfig_whenAccessing(zoneName, providerName, appName, resourceName, additionalResourceName1, additionalResourceName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "zone_id", "keycard_zone.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "application_id", "keycard_application.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "resource_id", "keycard_resource.test", "id"),
					resource.TestCheckResourceAttr("keycard_application_dependency.test", "when_accessing.#", "2"),
					resource.TestCheckTypeSetElemAttrPair("keycard_application_dependency.test", "when_accessing.*", "keycard_resource.additional1", "id"),
					resource.TestCheckTypeSetElemAttrPair("keycard_application_dependency.test", "when_accessing.*", "keycard_resource.additional2", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "keycard_application_dependency.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccApplicationDependencyImportStateIdFunc("keycard_application_dependency.test"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "resource_id",
			},
		},
	})
}

func TestAccApplicationDependencyResource_whenAccessingSingle(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")
	resourceName := acctest.RandomWithPrefix("tftest-resource")
	additionalResourceName := acctest.RandomWithPrefix("tftest-additional")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create dependency with single when_accessing
			{
				Config: testAccApplicationDependencyResourceConfig_whenAccessingSingle(zoneName, providerName, appName, resourceName, additionalResourceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "zone_id", "keycard_zone.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "application_id", "keycard_application.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "resource_id", "keycard_resource.test", "id"),
					resource.TestCheckResourceAttr("keycard_application_dependency.test", "when_accessing.#", "1"),
					resource.TestCheckResourceAttrPair("keycard_application_dependency.test", "when_accessing.0", "keycard_resource.additional", "id"),
				),
			},
		},
	})
}

// Helper function to generate import state ID in format zones/{zone-id}/applications/{application-id}/dependencies/{resource-id}.
func testAccApplicationDependencyImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		applicationID := rs.Primary.Attributes["application_id"]
		resourceID := rs.Primary.Attributes["resource_id"]

		if zoneID == "" || applicationID == "" || resourceID == "" {
			return "", fmt.Errorf("zone_id, application_id, or resource_id is empty")
		}

		return fmt.Sprintf("zones/%s/applications/%s/dependencies/%s", zoneID, applicationID, resourceID), nil
	}
}

func testAccApplicationDependencyResourceConfig_basic(zoneName, providerName, appName, resourceName string) string {
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
}

resource "keycard_application_dependency" "test" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  resource_id    = keycard_resource.test.id
}
`, zoneName, providerName, appName, resourceName)
}

func testAccApplicationDependencyResourceConfig_multiple(zoneName, providerName, appName, resourceName1, resourceName2 string) string {
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

resource "keycard_resource" "test1" {
  name                   = %[4]q
  identifier             = "https://%[4]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
}

resource "keycard_resource" "test2" {
  name                   = %[5]q
  identifier             = "https://%[5]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
}

resource "keycard_application_dependency" "test1" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  resource_id    = keycard_resource.test1.id
}

resource "keycard_application_dependency" "test2" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  resource_id    = keycard_resource.test2.id
}
`, zoneName, providerName, appName, resourceName1, resourceName2)
}

func testAccApplicationDependencyResourceConfig_withTwoResources(zoneName, providerName, appName, resourceName1, resourceName2, targetResource string) string {
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

resource "keycard_resource" "test1" {
  name                   = %[4]q
  identifier             = "https://%[4]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
}

resource "keycard_resource" "test2" {
  name                   = %[5]q
  identifier             = "https://%[5]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
}

resource "keycard_application_dependency" "test" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  resource_id    = keycard_resource.%[6]s.id
}
`, zoneName, providerName, appName, resourceName1, resourceName2, targetResource)
}

func testAccApplicationDependencyResourceConfig_whenAccessing(zoneName, providerName, appName, resourceName, additionalResourceName1, additionalResourceName2 string) string {
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
}

resource "keycard_resource" "additional1" {
  name                   = %[5]q
  identifier             = "https://%[5]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
  application_id         = keycard_application.test.id
}

resource "keycard_resource" "additional2" {
  name                   = %[6]q
  identifier             = "https://%[6]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
  application_id         = keycard_application.test.id
}

resource "keycard_application_dependency" "test" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  resource_id    = keycard_resource.test.id
  when_accessing = [
    keycard_resource.additional1.id,
    keycard_resource.additional2.id,
  ]
}
`, zoneName, providerName, appName, resourceName, additionalResourceName1, additionalResourceName2)
}

func testAccApplicationDependencyResourceConfig_whenAccessingSingle(zoneName, providerName, appName, resourceName, additionalResourceName string) string {
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
}

resource "keycard_resource" "additional" {
  name                   = %[5]q
  identifier             = "https://%[5]s.example.com"
  zone_id                = keycard_zone.test.id
  credential_provider_id = keycard_provider.test.id
  application_id         = keycard_application.test.id
}

resource "keycard_application_dependency" "test" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  resource_id    = keycard_resource.test.id
  when_accessing = [
    keycard_resource.additional.id,
  ]
}
`, zoneName, providerName, appName, resourceName, additionalResourceName)
}
