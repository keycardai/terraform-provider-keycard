package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationDataSource_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationDataSourceConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.keycard_organization.test", "id"),
					resource.TestCheckResourceAttrSet("data.keycard_organization.test", "name"),
					resource.TestCheckResourceAttrSet("data.keycard_organization.test", "label"),
					resource.TestCheckResourceAttrSet("data.keycard_organization.test", "zone_id"),
					resource.TestCheckResourceAttrSet("data.keycard_organization.test", "sso_enabled"),
				),
			},
		},
	})
}

func testAccOrganizationDataSourceConfig_basic() string {
	return `
data "keycard_organization" "test" {}
`
}
