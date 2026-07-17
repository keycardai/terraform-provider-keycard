package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicySetsDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetsDataSourceConfig(zoneName, rName),
				// No exact count: zones are pre-provisioned with platform-managed
				// policy sets, so only the presence of the created ones is stable.
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("data.keycard_policy_sets.test", "policy_sets.*", map[string]string{
						"name":        rName + "-a",
						"target_type": "zone",
						"active":      "false",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.keycard_policy_sets.test", "policy_sets.*", map[string]string{
						"name": rName + "-b",
					}),
				),
			},
		},
	})
}

func TestAccPolicySetsDataSource_nameFilter(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetsDataSourceConfig_nameFilter(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_policy_sets.test", "policy_sets.#", "1"),
					resource.TestCheckResourceAttr("data.keycard_policy_sets.test", "policy_sets.0.name", rName+"-a"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_sets.test", "policy_sets.0.id"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_sets.test", "policy_sets.0.owner_type"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_sets.test", "policy_sets.0.created_at"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_sets.test", "policy_sets.0.updated_at"),
				),
			},
		},
	})
}

func testAccPolicySetsDataSourceConfig(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy_set" "a" {
  name    = "%[2]s-a"
  zone_id = keycard_zone.test.id
}

resource "keycard_policy_set" "b" {
  name    = "%[2]s-b"
  zone_id = keycard_zone.test.id
}

data "keycard_policy_sets" "test" {
  zone_id = keycard_zone.test.id

  depends_on = [keycard_policy_set.a, keycard_policy_set.b]
}
`, zoneName, name)
}

func testAccPolicySetsDataSourceConfig_nameFilter(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy_set" "a" {
  name    = "%[2]s-a"
  zone_id = keycard_zone.test.id
}

resource "keycard_policy_set" "b" {
  name    = "%[2]s-b"
  zone_id = keycard_zone.test.id
}

data "keycard_policy_sets" "test" {
  zone_id = keycard_zone.test.id
  name    = "%[2]s-a"

  depends_on = [keycard_policy_set.a, keycard_policy_set.b]
}
`, zoneName, name)
}
