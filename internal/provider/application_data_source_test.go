package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application resource and fetch it with the data source
			{
				Config: testAccApplicationDataSourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match the resource
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "id",
						"keycard_application.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "zone_id",
						"keycard_application.test", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "name",
						"keycard_application.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "identifier",
						"keycard_application.test", "identifier",
					),
					resource.TestCheckResourceAttr("data.keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_application.test", "identifier", "https://"+rName+".example.com"),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	description := "Test application description"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with description and fetch it
			{
				Config: testAccApplicationDataSourceConfig_withDescription(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "id",
						"keycard_application.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "description",
						"keycard_application.test", "description",
					),
					resource.TestCheckResourceAttr("data.keycard_application.test", "description", description),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_withMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with metadata and fetch it
			{
				Config: testAccApplicationDataSourceConfig_withMetadata(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "id",
						"keycard_application.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "metadata.docs_url",
						"keycard_application.test", "metadata.docs_url",
					),
					resource.TestCheckResourceAttr("data.keycard_application.test", "metadata.docs_url", "https://docs.example.com/app"),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_withOAuth2(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with OAuth2 configuration and fetch it
			{
				Config: testAccApplicationDataSourceConfig_withOAuth2(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "id",
						"keycard_application.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "oauth2.redirect_uris.#",
						"keycard_application.test", "oauth2.redirect_uris.#",
					),
					resource.TestCheckResourceAttr("data.keycard_application.test", "oauth2.redirect_uris.#", "2"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "oauth2.redirect_uris.0", "https://"+rName+".example.com/callback"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "oauth2.redirect_uris.1", "https://"+rName+".example.com/auth/callback"),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_withTraits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with traits and fetch it
			{
				Config: testAccApplicationDataSourceConfig_withTraits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "id",
						"keycard_application.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "traits.#",
						"keycard_application.test", "traits.#",
					),
					resource.TestCheckResourceAttr("data.keycard_application.test", "traits.#", "1"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "traits.0", "gateway"),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_withConsent(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationDataSourceConfig_withConsent(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "consent",
						"keycard_application.test", "consent",
					),
					resource.TestCheckResourceAttr("data.keycard_application.test", "consent", "implicit"),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with all fields and fetch it
			{
				Config: testAccApplicationDataSourceConfig_complete(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "id",
						"keycard_application.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "name",
						"keycard_application.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "consent",
						"keycard_application.test", "consent",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "description",
						"keycard_application.test", "description",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "identifier",
						"keycard_application.test", "identifier",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "metadata.docs_url",
						"keycard_application.test", "metadata.docs_url",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "oauth2.redirect_uris.#",
						"keycard_application.test", "oauth2.redirect_uris.#",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application.test", "traits.#",
						"keycard_application.test", "traits.#",
					),
					resource.TestCheckResourceAttr("data.keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_application.test", "consent", "implicit"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "description", "Complete application with all fields"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "metadata.docs_url", "https://docs.example.com/complete"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "oauth2.redirect_uris.#", "2"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "traits.#", "1"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "traits.0", "gateway"),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_notFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch an application that doesn't exist
			{
				Config:      testAccApplicationDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile("Application Not Found"),
			},
		},
	})
}

func testAccApplicationDataSourceConfig_basic(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, appName)
}

func testAccApplicationDataSourceConfig_withDescription(appName, description string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name        = %[1]q
  description = %[2]q
  identifier  = "https://%[1]s.example.com"
  zone_id     = data.keycard_organization.test.zone_id
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, appName, description)
}

func testAccApplicationDataSourceConfig_withMetadata(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id

  metadata = {
    docs_url = "https://docs.example.com/app"
  }
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, appName)
}

func testAccApplicationDataSourceConfig_withOAuth2(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id

  oauth2 = {
    redirect_uris = [
      "https://%[1]s.example.com/callback",
      "https://%[1]s.example.com/auth/callback"
    ]
  }
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, appName)
}

func testAccApplicationDataSourceConfig_withTraits(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id

  traits = ["gateway"]
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, appName)
}

func testAccApplicationDataSourceConfig_withConsent(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  consent    = "implicit"
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, appName)
}

func testAccApplicationDataSourceConfig_complete(appName string) string {
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

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, appName)
}

func testAccApplicationDataSourceConfig_notFound() string {
	return testAccOrgZone + `
data "keycard_application" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "non-existent-application-id-12345"
}
`
}
