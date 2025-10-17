package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

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
					// Verify OAuth2 values are populated by the API
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.pkce_required"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.dcr_enabled"),
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
					// Verify OAuth2 values are still present after update
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.pkce_required"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.dcr_enabled"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccZoneResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

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
				),
			},
			// Update description
			{
				Config: testAccZoneResourceConfig_withDescription(rName, "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "description", "Updated description"),
				),
			},
			// Remove description
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_zone.test", "description"),
				),
			},
		},
	})
}

func TestAccZoneResource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

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
					// Verify OAuth2 values are set by the API (computed)
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.pkce_required"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "oauth2.dcr_enabled"),
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
