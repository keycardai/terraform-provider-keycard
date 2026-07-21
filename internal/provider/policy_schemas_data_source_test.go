package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicySchemasDataSource_basic(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySchemasDataSourceConfig(zoneName),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySchemasDataSourceConfig(zoneName),
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
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySchemasDataSourceConfig_isDefault(zoneName),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySchemasDataSourceConfig_isDefault(zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_policy_schemas.test", "schemas.#", "1"),
					resource.TestCheckResourceAttr("data.keycard_policy_schemas.test", "schemas.0.is_default", "true"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_schemas.test", "schemas.0.version", "data.keycard_policy_schema.default", "version"),
				),
			},
		},
	})
}

func testAccPolicySchemasDataSourceConfig(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_policy_schemas" "test" {
  zone_id = keycard_zone.test.id
}
`, zoneName)
}

func testAccPolicySchemasDataSourceConfig_isDefault(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_policy_schema" "default" {
  zone_id = keycard_zone.test.id
}

data "keycard_policy_schemas" "test" {
  zone_id    = keycard_zone.test.id
  is_default = true
}
`, zoneName)
}
