package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccGroupMemberUserID returns the user to add to test groups. The provider
// has no way to create a user, so the ID has to come from the environment. The
// user must belong to the organization's builtin zone, where these tests create
// their groups.
func testAccGroupMemberUserID(t *testing.T) string {
	userID := os.Getenv("KEYCARD_TEST_USER_ID")
	if userID == "" {
		t.Skip("KEYCARD_TEST_USER_ID must be set to run group membership tests")
	}

	return userID
}

func TestAccGroupMemberResource_basic(t *testing.T) {
	userID := testAccGroupMemberUserID(t)
	groupName := acctest.RandomWithPrefix("tftest-group")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupMemberResourceConfig_basic(groupName, userID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_group_member.test", "zone_id", testAccOrgZoneRef, "zone_id"),
					resource.TestCheckResourceAttrPair("keycard_group_member.test", "group_id", "keycard_group.test", "id"),
					resource.TestCheckResourceAttr("keycard_group_member.test", "user_id", userID),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "keycard_group_member.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccGroupMemberImportStateIdFunc("keycard_group_member.test"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "user_id",
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccGroupMemberResource_groupChange(t *testing.T) {
	userID := testAccGroupMemberUserID(t)
	groupName1 := acctest.RandomWithPrefix("tftest-group1")
	groupName2 := acctest.RandomWithPrefix("tftest-group2")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Add the user to the first group
			{
				Config: testAccGroupMemberResourceConfig_withTwoGroups(groupName1, groupName2, userID, "test1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_group_member.test", "group_id", "keycard_group.test1", "id"),
				),
			},
			// Move to the second group (should force replacement)
			{
				Config: testAccGroupMemberResourceConfig_withTwoGroups(groupName1, groupName2, userID, "test2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_group_member.test", "group_id", "keycard_group.test2", "id"),
				),
			},
		},
	})
}

// Helper function to generate import state ID in format zones/{zone-id}/groups/{group-id}/members/{user-id}.
func testAccGroupMemberImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		groupID := rs.Primary.Attributes["group_id"]
		userID := rs.Primary.Attributes["user_id"]

		if zoneID == "" || groupID == "" || userID == "" {
			return "", fmt.Errorf("zone_id, group_id, or user_id is empty")
		}

		return fmt.Sprintf("zones/%s/groups/%s/members/%s", zoneID, groupID, userID), nil
	}
}

func testAccGroupMemberResourceConfig_basic(groupName, userID string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_group" "test" {
  zone_id = data.keycard_organization.test.zone_id
  name    = %[1]q
}

resource "keycard_group_member" "test" {
  zone_id  = data.keycard_organization.test.zone_id
  group_id = keycard_group.test.id
  user_id  = %[2]q
}
`, groupName, userID)
}

func testAccGroupMemberResourceConfig_withTwoGroups(groupName1, groupName2, userID, groupRef string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_group" "test1" {
  zone_id = data.keycard_organization.test.zone_id
  name    = %[1]q
}

resource "keycard_group" "test2" {
  zone_id = data.keycard_organization.test.zone_id
  name    = %[2]q
}

resource "keycard_group_member" "test" {
  zone_id  = data.keycard_organization.test.zone_id
  group_id = keycard_group.%[4]s.id
  user_id  = %[3]q
}
`, groupName1, groupName2, userID, groupRef)
}
