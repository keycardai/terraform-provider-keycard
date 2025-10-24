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
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application resource and fetch it with the data source
			{
				Config: testAccApplicationDataSourceConfig_basic(zoneName, rName),
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
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	description := "Test application description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with description and fetch it
			{
				Config: testAccApplicationDataSourceConfig_withDescription(zoneName, rName, description),
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
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with metadata and fetch it
			{
				Config: testAccApplicationDataSourceConfig_withMetadata(zoneName, rName),
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
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with OAuth2 configuration and fetch it
			{
				Config: testAccApplicationDataSourceConfig_withOAuth2(zoneName, rName),
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

func TestAccApplicationDataSource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an application with all fields and fetch it
			{
				Config: testAccApplicationDataSourceConfig_complete(zoneName, rName),
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
					resource.TestCheckResourceAttr("data.keycard_application.test", "name", rName),
					resource.TestCheckResourceAttr("data.keycard_application.test", "description", "Complete application with all fields"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "metadata.docs_url", "https://docs.example.com/complete"),
					resource.TestCheckResourceAttr("data.keycard_application.test", "oauth2.redirect_uris.#", "2"),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_notFound(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone but attempt to fetch an application that doesn't exist
			{
				Config:      testAccApplicationDataSourceConfig_notFound(zoneName),
				ExpectError: regexp.MustCompile("Application Not Found"),
			},
		},
	})
}

func testAccApplicationDataSourceConfig_basic(zoneName, appName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, zoneName, appName)
}

func testAccApplicationDataSourceConfig_withDescription(zoneName, appName, description string) string {
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

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, zoneName, appName, description)
}

func testAccApplicationDataSourceConfig_withMetadata(zoneName, appName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id

  metadata = {
    docs_url = "https://docs.example.com/app"
  }
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, zoneName, appName)
}

func testAccApplicationDataSourceConfig_withOAuth2(zoneName, appName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id

  oauth2 = {
    redirect_uris = [
      "https://%[2]s.example.com/callback",
      "https://%[2]s.example.com/auth/callback"
    ]
  }
}

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, zoneName, appName)
}

func testAccApplicationDataSourceConfig_complete(zoneName, appName string) string {
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

data "keycard_application" "test" {
  zone_id = keycard_application.test.zone_id
  id      = keycard_application.test.id
}
`, zoneName, appName)
}

func testAccApplicationDataSourceConfig_notFound(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_application" "test" {
  zone_id = keycard_zone.test.id
  id      = "non-existent-application-id-12345"
}
`, zoneName)
}
