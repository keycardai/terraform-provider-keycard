package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const zoneResourceName = "keycard_zone.test"

// testAccCheckZoneIDSaved saves the zone ID to a variable for later comparison.
func testAccCheckZoneIDSaved(id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[zoneResourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", zoneResourceName)
		}
		*id = rs.Primary.ID
		return nil
	}
}

// testAccCheckZoneIDUnchanged verifies the zone ID hasn't changed (for updates that should NOT trigger replacement).
func testAccCheckZoneIDUnchanged(originalID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[zoneResourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", zoneResourceName)
		}
		if rs.Primary.ID != *originalID {
			return fmt.Errorf("Zone was replaced - ID changed from %s to %s", *originalID, rs.Primary.ID)
		}
		return nil
	}
}

func TestAccZoneResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	var zoneID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "id"),
					testAccCheckZoneIDSaved(&zoneID),
					// Verify OAuth2 values are populated by the API
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.pkce_required"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.dcr_enabled"),
					// Verify OAuth2 protocol URIs are populated by the API
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.issuer_uri"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.redirect_uri"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccZoneResourceConfig_basic(rName + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName+"-updated"),
					testAccCheckZoneIDUnchanged(&zoneID),
					// Verify OAuth2 values are still present after update
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.pkce_required"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.dcr_enabled"),
					// Verify OAuth2 protocol URIs remain stable after update
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.issuer_uri"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.redirect_uri"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccZoneResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	var zoneID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccZoneResourceConfig_withDescription(rName, "Test description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "description", "Test description"),
					testAccCheckZoneIDSaved(&zoneID),
				),
			},
			// Update description
			{
				Config: testAccZoneResourceConfig_withDescription(rName, "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "description", "Updated description"),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
			// Remove description
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_zone.test", "description"),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
		},
	})
}

func TestAccZoneResource_emptyDescriptionInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccZoneResourceConfig_withDescription(rName, ""),
				ExpectError: regexp.MustCompile(`Attribute description string length must be at least 1`),
			},
		},
	})
}

func TestAccZoneResource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	var zoneID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all fields
			{
				Config: testAccZoneResourceConfig_withDescription(rName, "Complete zone"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "description", "Complete zone"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "id"),
					testAccCheckZoneIDSaved(&zoneID),
					// Verify OAuth2 values are set by the API (computed)
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.pkce_required"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.dcr_enabled"),
					// Verify OAuth2 protocol URIs are set by the API (computed)
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.issuer_uri"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.redirect_uri"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccZoneResource_oauth2Custom(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	var zoneID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with custom OAuth2 settings
			{
				Config: testAccZoneResourceConfig_withOAuth2(rName, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "false"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "false"),
					testAccCheckZoneIDSaved(&zoneID),
					// Verify OAuth2 protocol URIs are populated regardless of settings
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.issuer_uri"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.redirect_uri"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccZoneResource_oauth2Updates(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	var zoneID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with OAuth2 disabled
			{
				Config: testAccZoneResourceConfig_withOAuth2(rName, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "false"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "false"),
					testAccCheckZoneIDSaved(&zoneID),
				),
			},
			// Update OAuth2 to enable PKCE only
			{
				Config: testAccZoneResourceConfig_withOAuth2(rName, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "true"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "false"),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
			// Update OAuth2 to enable both
			{
				Config: testAccZoneResourceConfig_withOAuth2(rName, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "true"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "true"),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
			// Update name and verify OAuth2 settings persist
			{
				Config: testAccZoneResourceConfig_withOAuth2(rName+"-updated", true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "true"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "true"),
					testAccCheckZoneIDUnchanged(&zoneID),
					// Verify OAuth2 protocol URIs persist through updates
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.issuer_uri"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.redirect_uri"),
				),
			},
		},
	})
}

func TestAccZoneResource_oauth2Defaults(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	var zoneID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without specifying OAuth2 block
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					// Verify defaults are applied (both should be true)
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "true"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "true"),
					testAccCheckZoneIDSaved(&zoneID),
				),
			},
			// Add OAuth2 block with explicit values
			{
				Config: testAccZoneResourceConfig_withOAuth2(rName, false, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "false"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "true"),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
			// Remove OAuth2 block (should retain last set values)
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Values should persist from previous state
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "false"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "true"),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
		},
	})
}

func testAccZoneResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}
`, name)
}

func testAccZoneResourceConfig_withDescription(name, description string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name        = %[1]q
  description = %[2]q
}
`, name, description)
}

func testAccZoneResourceConfig_withOAuth2(name string, pkceRequired, dcrEnabled bool) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q

  oauth2 = {
    pkce_required = %[2]t
    dcr_enabled   = %[3]t
  }
}
`, name, pkceRequired, dcrEnabled)
}

func TestAccZoneResource_encryptionKeyUpdate(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	kmsArn1 := os.Getenv("KEYCARD_TEST_KMS_KEY_1")
	kmsArn2 := os.Getenv("KEYCARD_TEST_KMS_KEY_2")
	var zoneID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first encryption_key
			{
				Config: testAccZoneResourceConfig_withEncryptionKey(rName, kmsArn1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "encryption_key.aws.arn", kmsArn1),
					testAccCheckZoneIDSaved(&zoneID),
				),
			},
			// Change encryption_key ARN - should update in-place
			{
				Config: testAccZoneResourceConfig_withEncryptionKey(rName, kmsArn2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "encryption_key.aws.arn", kmsArn2),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
		},
	})
}

func TestAccZoneResource_encryptionKeyAddRemove(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	kmsArn := os.Getenv("KEYCARD_TEST_KMS_KEY_1")
	var zoneID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without encryption_key
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckNoResourceAttr("keycard_zone.test", "encryption_key.aws.arn"),
					testAccCheckZoneIDSaved(&zoneID),
				),
			},
			// Add encryption_key - should update in-place
			{
				Config: testAccZoneResourceConfig_withEncryptionKey(rName, kmsArn),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "encryption_key.aws.arn", kmsArn),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
			// Remove encryption_key - should update in-place
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckNoResourceAttr("keycard_zone.test", "encryption_key.aws.arn"),
					testAccCheckZoneIDUnchanged(&zoneID),
				),
			},
		},
	})
}

func testAccZoneResourceConfig_withEncryptionKey(name, kmsArn string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q

  encryption_key = {
    aws = {
      arn = %[2]q
    }
  }
}
`, name, kmsArn)
}
