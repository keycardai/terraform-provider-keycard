package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationPublicKeyCredentialResource_identifier(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	jwksURI := fmt.Sprintf("https://%s.example.com/jwks.json", rName)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_basic(rName, jwksURI) + fmt.Sprintf(`
resource "keycard_application_public_key_credential" "custom" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  jwks_uri       = %q
  identifier     = %q
}
`, jwksURI, rName),
				Check: resource.TestCheckResourceAttr("keycard_application_public_key_credential.custom", "identifier", rName),
			},
		},
	})
}

func TestAccApplicationPublicKeyCredentialResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	urlValue := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_basic(rName, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "identifier"),
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "application_id"),
					resource.TestCheckResourceAttr("keycard_application_public_key_credential.test", "jwks_uri", urlValue),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_key_credential.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_key_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
			{
				ResourceName:      "keycard_application_public_key_credential.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["keycard_application_public_key_credential.test"]
					if !ok {
						return "", fmt.Errorf("Not found: keycard_application_public_key_credential.test")
					}
					zoneID := rs.Primary.Attributes["zone_id"]
					id := rs.Primary.ID
					return fmt.Sprintf("zones/%s/application-credentials/%s", zoneID, id), nil
				},
			},
		},
	})
}

func TestAccApplicationPublicKeyCredentialResource_urlChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	urlValue1 := fmt.Sprintf("https://%s-1.example.com", rName)
	urlValue2 := fmt.Sprintf("https://%s-2.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_basic(rName, urlValue1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "id"),
					resource.TestCheckResourceAttr("keycard_application_public_key_credential.test", "jwks_uri", urlValue1),
				),
			},
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_basic(rName, urlValue2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "id"),
					resource.TestCheckResourceAttr("keycard_application_public_key_credential.test", "jwks_uri", urlValue2),
				),
			},
		},
	})
}

func TestAccApplicationPublicKeyCredentialResource_applicationChange(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tftest-app1")
	rName2 := acctest.RandomWithPrefix("tftest-app2")
	urlValue := fmt.Sprintf("https://%s.example.com", rName1)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_basic(rName1, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_key_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_basic(rName2, urlValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_key_credential.test", "application_id",
						"keycard_application.test", "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationPublicKeyCredentialResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_inZone(rName, zoneName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_key_credential.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
					),
				),
			},
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_inZone(rName, zoneName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_key_credential.test", "zone_id",
						testAccOtherZoneRef, "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationPublicKeyCredentialResource_multipleCredentials(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	urlValue1 := fmt.Sprintf("https://%s-1.example.com", rName)
	urlValue2 := fmt.Sprintf("https://%s-2.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationPublicKeyCredentialResourceConfig_multiple(rName, urlValue1, urlValue2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test1", "id"),
					resource.TestCheckResourceAttr("keycard_application_public_key_credential.test1", "jwks_uri", urlValue1),
					resource.TestCheckResourceAttrSet("keycard_application_public_key_credential.test2", "id"),
					resource.TestCheckResourceAttr("keycard_application_public_key_credential.test2", "jwks_uri", urlValue2),
					resource.TestCheckResourceAttrPair(
						"keycard_application_public_key_credential.test1", "application_id",
						"keycard_application_public_key_credential.test2", "application_id",
					),
				),
			},
		},
	})
}

func testAccApplicationPublicKeyCredentialResourceConfig_basic(appName, urlValue string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_public_key_credential" "test" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  jwks_uri       = %[2]q
}
`, appName, urlValue)
}

func testAccApplicationPublicKeyCredentialResourceConfig_multiple(appName, urlValue1, urlValue2 string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_public_key_credential" "test1" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  jwks_uri       = %[2]q
}

resource "keycard_application_public_key_credential" "test2" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  jwks_uri       = %[3]q
}
`, appName, urlValue1, urlValue2)
}

func testAccApplicationPublicKeyCredentialResourceConfig_inZone(appName, zoneName string, otherZone bool) string {
	zoneConfig, zoneID := testAccZone(otherZone, zoneName)

	return zoneConfig + fmt.Sprintf(`
resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = %[2]s
}

resource "keycard_application_public_key_credential" "test" {
  zone_id        = %[2]s
  application_id = keycard_application.test.id
  jwks_uri       = "https://%[1]s.example.com"
}
`, appName, zoneID)
}
