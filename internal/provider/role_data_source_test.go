package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoleDataSource_byIdentifier(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceConfig_byIdentifier(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.keycard_role.by_identifier", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_role.by_identifier", "zone_id", "data.keycard_organization.test", "zone_id"),
					resource.TestCheckResourceAttr("data.keycard_role.by_identifier", "identifier", "viewer"),
					resource.TestCheckResourceAttr("data.keycard_role.by_identifier", "owner_type", "platform"),
					resource.TestCheckResourceAttrSet("data.keycard_role.by_identifier", "created_at"),
					resource.TestCheckResourceAttrSet("data.keycard_role.by_identifier", "updated_at"),
				),
			},
		},
	})
}

func TestAccRoleDataSource_byID(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceConfig_byID(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_role.by_id", "id", "data.keycard_role.by_identifier", "id"),
					resource.TestCheckResourceAttr("data.keycard_role.by_id", "identifier", "viewer"),
					resource.TestCheckResourceAttr("data.keycard_role.by_id", "owner_type", "platform"),
				),
			},
		},
	})
}

func TestAccRoleDataSource_notFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile(`Role Not Found`),
			},
		},
	})
}

func TestAccRoleDataSource_ownerTypeRequiredWithIdentifier(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleDataSourceConfig_identifierWithoutOwnerType(),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

// The organization zone is the only zone with roles: custom zones have none by
// default and the provider cannot create them yet.
const testAccRoleOrgZone = `
data "keycard_organization" "test" {}
`

func testAccRoleDataSourceConfig_byIdentifier() string {
	return testAccRoleOrgZone + `
data "keycard_role" "by_identifier" {
  zone_id    = data.keycard_organization.test.zone_id
  identifier = "viewer"
  owner_type = "platform"
}
`
}

func testAccRoleDataSourceConfig_byID() string {
	return testAccRoleDataSourceConfig_byIdentifier() + `
data "keycard_role" "by_id" {
  zone_id = data.keycard_organization.test.zone_id
  id      = data.keycard_role.by_identifier.id
}
`
}

func testAccRoleDataSourceConfig_notFound() string {
	return testAccRoleOrgZone + `
data "keycard_role" "missing" {
  zone_id    = data.keycard_organization.test.zone_id
  identifier = "no-such-role"
  owner_type = "platform"
}
`
}

func testAccRoleDataSourceConfig_identifierWithoutOwnerType() string {
	return testAccRoleOrgZone + `
data "keycard_role" "missing_owner_type" {
  zone_id    = data.keycard_organization.test.zone_id
  identifier = "viewer"
}
`
}
