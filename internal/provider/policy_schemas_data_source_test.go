package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicySchemasDataSource_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySchemasDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.keycard_policy_schemas.test", "schemas.#"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_schemas.test", "schemas.0.version"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_schemas.test", "schemas.0.status"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_schemas.test", "schemas.0.cedar_schema"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_schemas.test", "schemas.0.created_at"),
				),
			},
		},
	})
}

func TestAccPolicySchemasDataSource_isDefault(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySchemasDataSourceConfig_isDefault(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_policy_schemas.test", "schemas.#", "1"),
					resource.TestCheckResourceAttr("data.keycard_policy_schemas.test", "schemas.0.is_default", "true"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_schemas.test", "schemas.0.version", "data.keycard_policy_schema.default", "version"),
				),
			},
		},
	})
}

func testAccPolicySchemasDataSourceConfig() string {
	return testAccOrgZone + `
data "keycard_policy_schemas" "test" {
  zone_id = data.keycard_organization.test.zone_id
}
`
}

func testAccPolicySchemasDataSourceConfig_isDefault() string {
	return testAccOrgZone + `
data "keycard_policy_schema" "default" {
  zone_id = data.keycard_organization.test.zone_id
}

data "keycard_policy_schemas" "test" {
  zone_id    = data.keycard_organization.test.zone_id
  is_default = true
}
`
}
