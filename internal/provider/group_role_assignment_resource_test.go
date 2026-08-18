package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccGroupRole is the role assigned throughout these tests. The provider
// exposes no way to create roles, so a built-in platform role is looked up with
// the keycard_role data source.
const (
	testAccGroupRoleIdentifier = "viewer"
	testAccGroupRoleOwnerType  = "platform"
)

func TestAccGroupRoleAssignmentResource_basic(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupRoleAssignmentResourceConfig_basic(groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_group_role_assignment.test", "zone_id", "data.keycard_organization.test", "zone_id"),
					resource.TestCheckResourceAttrPair("keycard_group_role_assignment.test", "group_id", "keycard_group.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_group_role_assignment.test", "principal_id", "keycard_group.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_group_role_assignment.test", "role_id", "data.keycard_role.test", "id"),
					resource.TestCheckResourceAttr("keycard_group_role_assignment.test", "principal_type", "group"),
					resource.TestCheckResourceAttrSet("keycard_group_role_assignment.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_group_role_assignment.test",
				ImportState:       true,
				ImportStateIdFunc: testAccGroupRoleAssignmentImportStateIdFunc("keycard_group_role_assignment.test"),
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccGroupRoleAssignmentResource_scoped(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupRoleAssignmentResourceConfig_scoped(groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_group_role_assignment.test", "scope_type", "zone"),
					resource.TestCheckResourceAttrPair("keycard_group_role_assignment.test", "scope_id", "data.keycard_organization.test", "zone_id"),
				),
			},
		},
	})
}

func TestAccGroupRoleAssignmentResource_scopeTypeWithoutScopeID(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGroupRoleAssignmentResourceConfig_scopeTypeWithoutScopeID(groupName),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

// Helper function to generate import state ID in format zones/{zone-id}/groups/{group-id}/roles/{assignment-id}.
func testAccGroupRoleAssignmentImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		groupID := rs.Primary.Attributes["group_id"]
		assignmentID := rs.Primary.Attributes["id"]

		if zoneID == "" || groupID == "" || assignmentID == "" {
			return "", fmt.Errorf("zone_id, group_id, or id is empty")
		}

		return fmt.Sprintf("zones/%s/groups/%s/roles/%s", zoneID, groupID, assignmentID), nil
	}
}

// testAccGroupRoleAssignmentConfigBase declares the org zone, the group to
// assign to, and the role looked up by identifier and owner type.
func testAccGroupRoleAssignmentConfigBase(groupName string) string {
	return fmt.Sprintf(`
data "keycard_organization" "test" {}

data "keycard_role" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  identifier = %[2]q
  owner_type = %[3]q
}

resource "keycard_group" "test" {
  zone_id = data.keycard_organization.test.zone_id
  name    = %[1]q
}
`, groupName, testAccGroupRoleIdentifier, testAccGroupRoleOwnerType)
}

func testAccGroupRoleAssignmentResourceConfig_basic(groupName string) string {
	return testAccGroupRoleAssignmentConfigBase(groupName) + `
resource "keycard_group_role_assignment" "test" {
  zone_id  = data.keycard_organization.test.zone_id
  group_id = keycard_group.test.id
  role_id  = data.keycard_role.test.id
}
`
}

func testAccGroupRoleAssignmentResourceConfig_scoped(groupName string) string {
	return testAccGroupRoleAssignmentConfigBase(groupName) + `
resource "keycard_group_role_assignment" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  group_id   = keycard_group.test.id
  role_id    = data.keycard_role.test.id
  scope_type = "zone"
  scope_id   = data.keycard_organization.test.zone_id
}
`
}

func testAccGroupRoleAssignmentResourceConfig_scopeTypeWithoutScopeID(groupName string) string {
	return testAccGroupRoleAssignmentConfigBase(groupName) + `
resource "keycard_group_role_assignment" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  group_id   = keycard_group.test.id
  role_id    = data.keycard_role.test.id
  scope_type = "zone"
}
`
}
