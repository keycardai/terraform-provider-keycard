package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicySetDataSource_byID(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetDataSourceConfig_byID(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_policy_set.test", "id", "keycard_policy_set.test", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_set.test", "name", "keycard_policy_set.test", "name"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_set.test", "target_type", "keycard_policy_set.test", "target_type"),
					resource.TestCheckResourceAttr("data.keycard_policy_set.test", "name", rName),
					resource.TestCheckResourceAttrSet("data.keycard_policy_set.test", "owner_type"),
					resource.TestCheckResourceAttr("data.keycard_policy_set.test", "active", "false"),
				),
			},
		},
	})
}

func TestAccPolicySetDataSource_byName(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetDataSourceConfig_byName(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_policy_set.test", "id", "keycard_policy_set.test", "id"),
					resource.TestCheckResourceAttr("data.keycard_policy_set.test", "name", rName),
				),
			},
		},
	})
}

func TestAccPolicySetDataSource_notFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicySetDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile(`No policy set found with name`),
			},
		},
	})
}

func TestAccPolicySetDataSource_idNotFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesShortRetry,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicySetDataSourceConfig_idNotFound(),
				ExpectError: regexp.MustCompile(`Policy Set Not Found`),
			},
		},
	})
}

func TestAccPolicySetDataSource_idAndNameConflict(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicySetDataSourceConfig_idAndName(),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func testAccPolicySetDataSourceConfig_byID(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy_set" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

data "keycard_policy_set" "test" {
  zone_id = keycard_zone.test.id
  id      = keycard_policy_set.test.id
}
`, zoneName, name)
}

func testAccPolicySetDataSourceConfig_byName(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy_set" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

data "keycard_policy_set" "test" {
  zone_id = keycard_zone.test.id
  name    = keycard_policy_set.test.name
}
`, zoneName, name)
}

func testAccPolicySetDataSourceConfig_notFound() string {
	return testAccOrgZone + `
data "keycard_policy_set" "test" {
  zone_id = data.keycard_organization.test.zone_id
  name    = "does-not-exist-policy-set"
}
`
}

func testAccPolicySetDataSourceConfig_idNotFound() string {
	return testAccOrgZone + `
data "keycard_policy_set" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "does-not-exist-policy-set-id"
}
`
}

func testAccPolicySetDataSourceConfig_idAndName() string {
	return testAccOrgZone + `
data "keycard_policy_set" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "some-id"
  name    = "some-name"
}
`
}
