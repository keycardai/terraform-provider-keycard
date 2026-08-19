package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicyDataSource_byID(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDataSourceConfig_byID(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_policy.test", "id", "keycard_policy.test", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_policy.test", "name", "keycard_policy.test", "name"),
					resource.TestCheckResourceAttrPair("data.keycard_policy.test", "description", "keycard_policy.test", "description"),
					resource.TestCheckResourceAttr("data.keycard_policy.test", "name", rName),
					resource.TestCheckResourceAttrSet("data.keycard_policy.test", "owner_type"),
					resource.TestCheckResourceAttrSet("data.keycard_policy.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.keycard_policy.test", "updated_at"),
				),
			},
		},
	})
}

func TestAccPolicyDataSource_byName(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDataSourceConfig_byName(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_policy.test", "id", "keycard_policy.test", "id"),
					resource.TestCheckResourceAttr("data.keycard_policy.test", "name", rName),
				),
			},
		},
	})
}

func TestAccPolicyDataSource_notFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicyDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile(`No policy found with name`),
			},
		},
	})
}

func TestAccPolicyDataSource_idNotFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesShortRetry,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicyDataSourceConfig_idNotFound(),
				ExpectError: regexp.MustCompile(`Policy Not Found`),
			},
		},
	})
}

func TestAccPolicyDataSource_idAndNameConflict(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicyDataSourceConfig_idAndName(),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func testAccPolicyDataSourceConfig_byID(zoneName, policyName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "test" {
  name        = %[2]q
  description = "data source by id"
  zone_id     = keycard_zone.test.id
}

data "keycard_policy" "test" {
  zone_id = keycard_zone.test.id
  id      = keycard_policy.test.id
}
`, zoneName, policyName)
}

func testAccPolicyDataSourceConfig_byName(zoneName, policyName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

data "keycard_policy" "test" {
  zone_id = keycard_zone.test.id
  name    = keycard_policy.test.name
}
`, zoneName, policyName)
}

func testAccPolicyDataSourceConfig_notFound() string {
	return testAccOrgZone + `
data "keycard_policy" "test" {
  zone_id = data.keycard_organization.test.zone_id
  name    = "does-not-exist-policy"
}
`
}

func testAccPolicyDataSourceConfig_idNotFound() string {
	return testAccOrgZone + `
data "keycard_policy" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "does-not-exist-policy-id"
}
`
}

func testAccPolicyDataSourceConfig_idAndName() string {
	return testAccOrgZone + `
data "keycard_policy" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "some-id"
  name    = "some-name"
}
`
}
