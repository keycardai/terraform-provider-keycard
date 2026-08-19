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
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource and fetch it with the data source
			{
				Config: testAccResourceDataSourceConfig_basic(providerName, rName),
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
					resource.TestCheckResourceAttrPair(
						"data.keycard_resource.test", "prefix",
						"keycard_resource.test", "prefix",
					),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "identifier", "https://"+rName+".example.com"),
					resource.TestCheckResourceAttr("data.keycard_resource.test", "prefix", "false"),
				),
			},
		},
	})
}

func TestAccResourceDataSource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	description := "Test resource description"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with description and fetch it
			{
				Config: testAccResourceDataSourceConfig_withDescription(providerName, rName, description),
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
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with metadata and fetch it
			{
				Config: testAccResourceDataSourceConfig_withMetadata(providerName, rName),
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
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with OAuth2 scopes and fetch it
			{
				Config: testAccResourceDataSourceConfig_withScopes(providerName, rName),
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
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with application_id and fetch it
			{
				Config: testAccResourceDataSourceConfig_withApplication(providerName, appName, rName),
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
	providerName := acctest.RandomWithPrefix("tftest-provider")
	appName := acctest.RandomWithPrefix("tftest-app")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource with all fields and fetch it
			{
				Config: testAccResourceDataSourceConfig_complete(providerName, appName, rName),
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
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch a resource that doesn't exist
			{
				Config:      testAccResourceDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile("Resource Not Found"),
			},
		},
	})
}

func TestAccResourceDataSource_byIdentifier(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	providerName := acctest.RandomWithPrefix("tftest-provider")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource and fetch it by identifier
			{
				Config: testAccResourceDataSourceConfig_byIdentifier(providerName, rName),
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
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch a resource by identifier that doesn't exist
			{
				Config:      testAccResourceDataSourceConfig_byIdentifier_notFound(),
				ExpectError: regexp.MustCompile("Resource Not Found"),
			},
		},
	})
}

func TestAccResourceDataSource_validation_bothIdAndIdentifier(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to provide both id and identifier
			{
				Config:      testAccResourceDataSourceConfig_bothIdAndIdentifier(),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}

func TestAccResourceDataSource_validation_neitherIdNorIdentifier(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to provide neither id nor identifier
			{
				Config:      testAccResourceDataSourceConfig_neitherIdNorIdentifier(),
				ExpectError: regexp.MustCompile("Missing Attribute Configuration"),
			},
		},
	})
}

func testAccResourceDataSourceConfig_basic(providerName, resourceName string) string {
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

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, providerName, resourceName)
}

func testAccResourceDataSourceConfig_withDescription(providerName, resourceName, description string) string {
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

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, providerName, resourceName, description)
}

func testAccResourceDataSourceConfig_withMetadata(providerName, resourceName string) string {
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
    docs_url = "https://docs.example.com/resource"
  }
}

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, providerName, resourceName)
}

func testAccResourceDataSourceConfig_withScopes(providerName, resourceName string) string {
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

  oauth2 = {
    scopes = ["read", "write"]
  }
}

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, providerName, resourceName)
}

func testAccResourceDataSourceConfig_withApplication(providerName, appName, resourceName string) string {
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

data "keycard_resource" "test" {
  zone_id = keycard_resource.test.zone_id
  id      = keycard_resource.test.id
}
`, providerName, appName, resourceName)
}

func testAccResourceDataSourceConfig_complete(providerName, appName, resourceName string) string {
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
`, providerName, appName, resourceName)
}

func testAccResourceDataSourceConfig_notFound() string {
	return testAccOrgZone + `
data "keycard_resource" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "non-existent-resource-id-12345"
}
`
}

func testAccResourceDataSourceConfig_byIdentifier(providerName, resourceName string) string {
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

data "keycard_resource" "test" {
  zone_id    = keycard_resource.test.zone_id
  identifier = keycard_resource.test.identifier
}
`, providerName, resourceName)
}

func testAccResourceDataSourceConfig_byIdentifier_notFound() string {
	return testAccOrgZone + `
data "keycard_resource" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  identifier = "https://non-existent-resource.example.com"
}
`
}

func testAccResourceDataSourceConfig_bothIdAndIdentifier() string {
	return testAccOrgZone + `
data "keycard_resource" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  id         = "some-id"
  identifier = "https://some-resource.example.com"
}
`
}

func testAccResourceDataSourceConfig_neitherIdNorIdentifier() string {
	return testAccOrgZone + `
data "keycard_resource" "test" {
  zone_id = data.keycard_organization.test.zone_id
}
`
}
