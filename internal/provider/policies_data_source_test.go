package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPoliciesDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoliciesDataSourceConfig(zoneName, rName),
				// No exact count: zones are pre-provisioned with platform-managed
				// policies, so only the presence of the created ones is stable.
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("data.keycard_policies.test", "policies.*", map[string]string{
						"name":        rName + "-a",
						"description": "policies data source a",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.keycard_policies.test", "policies.*", map[string]string{
						"name": rName + "-b",
					}),
				),
			},
		},
	})
}

func TestAccPoliciesDataSource_nameFilter(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoliciesDataSourceConfig_nameFilter(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_policies.test", "policies.#", "1"),
					resource.TestCheckResourceAttr("data.keycard_policies.test", "policies.0.name", rName+"-a"),
					resource.TestCheckResourceAttrSet("data.keycard_policies.test", "policies.0.id"),
					resource.TestCheckResourceAttrSet("data.keycard_policies.test", "policies.0.owner_type"),
					resource.TestCheckResourceAttrSet("data.keycard_policies.test", "policies.0.created_at"),
					resource.TestCheckResourceAttrSet("data.keycard_policies.test", "policies.0.updated_at"),
				),
			},
		},
	})
}

func testAccPoliciesDataSourceConfig(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "a" {
  name        = "%[2]s-a"
  description = "policies data source a"
  zone_id     = keycard_zone.test.id
}

resource "keycard_policy" "b" {
  name    = "%[2]s-b"
  zone_id = keycard_zone.test.id
}

data "keycard_policies" "test" {
  zone_id = keycard_zone.test.id

  depends_on = [keycard_policy.a, keycard_policy.b]
}
`, zoneName, name)
}

func testAccPoliciesDataSourceConfig_nameFilter(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "a" {
  name    = "%[2]s-a"
  zone_id = keycard_zone.test.id
}

resource "keycard_policy" "b" {
  name    = "%[2]s-b"
  zone_id = keycard_zone.test.id
}

data "keycard_policies" "test" {
  zone_id = keycard_zone.test.id
  name    = "%[2]s-a"

  depends_on = [keycard_policy.a, keycard_policy.b]
}
`, zoneName, name)
}
