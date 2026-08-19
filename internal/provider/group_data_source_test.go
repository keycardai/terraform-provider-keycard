package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupDataSource_byID(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")
	identifier := acctest.RandomWithPrefix("tftest-identifier")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig_byID(groupName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_group.test", "id", "keycard_group.test", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_group.test", "zone_id", testAccOrgZoneRef, "zone_id"),
					resource.TestCheckResourceAttr("data.keycard_group.test", "name", groupName),
					resource.TestCheckResourceAttr("data.keycard_group.test", "identifier", identifier),
				),
			},
		},
	})
}

func TestAccGroupDataSource_byIdentifier(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")
	identifier := acctest.RandomWithPrefix("tftest-identifier")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig_byIdentifier(groupName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_group.test", "id", "keycard_group.test", "id"),
					resource.TestCheckResourceAttr("data.keycard_group.test", "name", groupName),
					resource.TestCheckResourceAttr("data.keycard_group.test", "identifier", identifier),
				),
			},
		},
	})
}

func TestAccGroupDataSource_bothIDAndIdentifier(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tftest-group")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGroupDataSourceConfig_bothIDAndIdentifier(groupName),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func testAccGroupDataSourceConfig_byID(groupName, identifier string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_group" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  name       = %[1]q
  identifier = %[2]q
}

data "keycard_group" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = keycard_group.test.id
}
`, groupName, identifier)
}

func testAccGroupDataSourceConfig_byIdentifier(groupName, identifier string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_group" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  name       = %[1]q
  identifier = %[2]q
}

data "keycard_group" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  identifier = keycard_group.test.identifier
}
`, groupName, identifier)
}

func testAccGroupDataSourceConfig_bothIDAndIdentifier(groupName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_group" "test" {
  zone_id = data.keycard_organization.test.zone_id
  name    = %[1]q
}

data "keycard_group" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  id         = keycard_group.test.id
  identifier = keycard_group.test.identifier
}
`, groupName)
}
