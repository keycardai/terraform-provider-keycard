package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGroupResource_basic(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupResourceConfig_basic(groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_group.test", "zone_id", testAccOrgZoneRef, "zone_id"),
					resource.TestCheckResourceAttr("keycard_group.test", "name", groupName),
					resource.TestCheckResourceAttrSet("keycard_group.test", "id"),
					// Not configured, so the API derives it from the name.
					resource.TestCheckResourceAttrSet("keycard_group.test", "identifier"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_group.test",
				ImportState:       true,
				ImportStateIdFunc: testAccGroupImportStateIdFunc("keycard_group.test"),
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccGroupResource_withIdentifier(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")
	identifier := acctest.RandomWithPrefix("tftest-identifier")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupResourceConfig_withIdentifier(groupName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_group.test", "name", groupName),
					resource.TestCheckResourceAttr("keycard_group.test", "identifier", identifier),
				),
			},
		},
	})
}

func TestAccGroupResource_update(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")
	updatedGroupName := acctest.RandomWithPrefix("tftest-group-updated")
	identifier := acctest.RandomWithPrefix("tftest-identifier")
	updatedIdentifier := acctest.RandomWithPrefix("tftest-identifier-updated")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupResourceConfig_withIdentifier(groupName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_group.test", "name", groupName),
					resource.TestCheckResourceAttr("keycard_group.test", "identifier", identifier),
				),
			},
			// Update the name and identifier in place
			{
				Config: testAccGroupResourceConfig_withIdentifier(updatedGroupName, updatedIdentifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_group.test", "name", updatedGroupName),
					resource.TestCheckResourceAttr("keycard_group.test", "identifier", updatedIdentifier),
				),
			},
		},
	})
}

// Helper function to generate import state ID in format zones/{zone-id}/groups/{group-id}.
func testAccGroupImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		groupID := rs.Primary.Attributes["id"]

		if zoneID == "" || groupID == "" {
			return "", fmt.Errorf("zone_id or id is empty")
		}

		return fmt.Sprintf("zones/%s/groups/%s", zoneID, groupID), nil
	}
}

func testAccGroupResourceConfig_basic(groupName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_group" "test" {
  zone_id = data.keycard_organization.test.zone_id
  name    = %[1]q
}
`, groupName)
}

func testAccGroupResourceConfig_withIdentifier(groupName, identifier string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_group" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  name       = %[1]q
  identifier = %[2]q
}
`, groupName, identifier)
}
