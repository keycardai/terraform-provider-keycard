package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource and fetch it with the data source
			{
				Config: testAccResourceDataSourceConfig_basic(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match the resource
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "id",
						"keycard_resource.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "zone_id",
						"keycard_resource.test", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "name",
						"keycard_resource.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "identifier",
						"keycard_resource.test", "identifier",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "credential_provider_id",
						"keycard_resource.test", "credential_provider_id",
					),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "identifier", "https://"+rName+".example.com"),
				),
			},
		},
	})
}

func TestAccResourceDataSource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	description := "Test resource description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with description and fetch it
			{
				Config: testAccResourceDataSourceConfig_withDescription(zoneName, providerName, rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "id",
						"keycard_resource.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "description",
						"keycard_resource.test", "description",
					),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "description", description),
				),
			},
		},
	})
}

func TestAccResourceDataSource_withMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with metadata and fetch it
			{
				Config: testAccResourceDataSourceConfig_withMetadata(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "id",
						"keycard_resource.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "metadata.docs_url",
						"keycard_resource.test", "metadata.docs_url",
					),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "metadata.docs_url", "https://docs.example.com/resource"),
				),
			},
		},
	})
}

func TestAccResourceDataSource_withScopes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with OAuth2 scopes and fetch it
			{
				Config: testAccResourceDataSourceConfig_withScopes(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "id",
						"keycard_resource.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "oauth2.scopes.#",
						"keycard_resource.test", "oauth2.scopes.#",
					),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "oauth2.scopes.#", "2"),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "oauth2.scopes.0", "read"),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "oauth2.scopes.1", "write"),
				),
			},
		},
	})
}

func TestAccResourceDataSource_withApplication(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with application_id and fetch it
			{
				Config: testAccResourceDataSourceConfig_withApplication(zoneName, providerName, appName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "id",
						"keycard_resource.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "application_id",
						"keycard_resource.test", "application_id",
					),
					resource.TestCheckResourceAttrSet("data.keycard_resource.test", "application_id"),
				),
			},
		},
	})
}

func TestAccResourceDataSource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with all fields and fetch it
			{
				Config: testAccResourceDataSourceConfig_complete(zoneName, providerName, appName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "id",
						"keycard_resource.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "name",
						"keycard_resource.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "description",
						"keycard_resource.test", "description",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "identifier",
						"keycard_resource.test", "identifier",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "metadata.docs_url",
						"keycard_resource.test", "metadata.docs_url",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "oauth2.scopes.#",
						"keycard_resource.test", "oauth2.scopes.#",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "application_id",
						"keycard_resource.test", "application_id",
					),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "description", "Complete resource with all fields"),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "metadata.docs_url", "https://docs.example.com/complete"),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "oauth2.scopes.#", "2"),
				),
			},
		},
	})
}

func TestAccResourceDataSource_notFound(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone but attempt to fetch a resource that doesn't exist
			{
				Config:      testAccResourceDataSourceConfig_notFound(zoneName),
				ExpectError: regexp.MustCompile("Resource Not Found"),
			},
		},
	})
}

func TestAccResourceDataSource_byIdentifier(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource and fetch it by identifier
			{
				Config: testAccResourceDataSourceConfig_byIdentifier(zoneName, providerName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match the resource
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "id",
						"keycard_resource.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "zone_id",
						"keycard_resource.test", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "name",
						"keycard_resource.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "identifier",
						"keycard_resource.test", "identifier",
					),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "identifier", "https://"+rName+".example.com"),
				),
			},
		},
	})
}

func TestAccResourceDataSource_byIdentifier_notFound(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch a resource by identifier that doesn't exist
			{
				Config:      testAccResourceDataSourceConfig_byIdentifier_notFound(zoneName),
				ExpectError: regexp.MustCompile("Resource Not Found"),
			},
		},
	})
}

func TestAccResourceDataSource_validation_bothIdAndIdentifier(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to provide both id and identifier
			{
				Config:      testAccResourceDataSourceConfig_bothIdAndIdentifier(zoneName),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}

func TestAccResourceDataSource_validation_neitherIdNorIdentifier(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to provide neither id nor identifier
			{
				Config:      testAccResourceDataSourceConfig_neitherIdNorIdentifier(zoneName),
				ExpectError: regexp.MustCompile("Missing Attribute Configuration"),
			},
		},
	})
}

func testAccResourceDataSourceConfig_basic(zoneName, providerName, resourceName string) string {
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

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, zoneName, providerName, resourceName)
}

func testAccResourceDataSourceConfig_withDescription(zoneName, providerName, resourceName, description string) string {
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

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, zoneName, providerName, resourceName, description)
}

func testAccResourceDataSourceConfig_withMetadata(zoneName, providerName, resourceName string) string {
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
    docs_url = "https://docs.example.com/resource"
  }
}

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, zoneName, providerName, resourceName)
}

func testAccResourceDataSourceConfig_withScopes(zoneName, providerName, resourceName string) string {
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

  oauth2 = {
    scopes = ["read", "write"]
  }
}

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, zoneName, providerName, resourceName)
}

func testAccResourceDataSourceConfig_withApplication(zoneName, providerName, appName, resourceName string) string {
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

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, zoneName, providerName, appName, resourceName)
}

func testAccResourceDataSourceConfig_complete(zoneName, providerName, appName, resourceName string) string {
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

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, zoneName, providerName, appName, resourceName)
}

func testAccResourceDataSourceConfig_notFound(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_resource" "test" {
  zone_id = keycard_zone.test.id
  id      = "non-existent-resource-id-12345"
}
`, zoneName)
}

func testAccResourceDataSourceConfig_byIdentifier(zoneName, providerName, resourceName string) string {
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

data "keycard_resource" "test" {
  zone_id    = keycard_resource.test.zone_id
  identifier = keycard_resource.test.identifier
}
`, zoneName, providerName, resourceName)
}

func testAccResourceDataSourceConfig_byIdentifier_notFound(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_resource" "test" {
  zone_id    = keycard_zone.test.id
  identifier = "https://non-existent-resource.example.com"
}
`, zoneName)
}

func testAccResourceDataSourceConfig_bothIdAndIdentifier(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_resource" "test" {
  zone_id    = keycard_zone.test.id
  id         = "some-id"
  identifier = "https://some-resource.example.com"
}
`, zoneName)
}

func testAccResourceDataSourceConfig_neitherIdNorIdentifier(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_resource" "test" {
  zone_id = keycard_zone.test.id
}
`, zoneName)
}
