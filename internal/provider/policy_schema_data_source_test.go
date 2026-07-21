package provider

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Zones are registered with the policy service asynchronously after
// creation, and the service negatively caches unknown zones for up to a
// minute. The data source retries through that, but tests run much faster
// when the first read lands after bootstrap instead of poisoning the cache.
func waitForZoneBootstrap() {
	time.Sleep(5 * time.Second)
}

func TestAccPolicySchemaDataSource_default(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneOnlyConfig(rName),
			},
			// Fetch the zone's default schema without specifying a version
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySchemaDataSourceConfig_default(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.keycard_policy_schema.test", "version"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_schema.test", "status"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_schema.test", "cedar_schema"),
					resource.TestCheckResourceAttr("data.keycard_policy_schema.test", "is_default", "true"),
				),
			},
		},
	})
}

func TestAccPolicySchemaDataSource_byVersion(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneOnlyConfig(rName),
			},
			// Look up the default schema, then fetch the same schema by version
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySchemaDataSourceConfig_byVersion(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_policy_schema.by_version", "version",
						"data.keycard_policy_schema.default", "version",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_policy_schema.by_version", "cedar_schema",
						"data.keycard_policy_schema.default", "cedar_schema",
					),
					resource.TestCheckResourceAttrSet("data.keycard_policy_schema.by_version", "status"),
				),
			},
		},
	})
}

func TestAccPolicySchemaDataSource_versionNotFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesShortRetry,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneOnlyConfig(rName),
			},
			// Attempt to fetch a schema version that doesn't exist
			{
				PreConfig:   waitForZoneBootstrap,
				Config:      testAccPolicySchemaDataSourceConfig_versionNotFound(rName),
				ExpectError: regexp.MustCompile("Policy Schema Not Found"),
			},
		},
	})
}

func testAccZoneOnlyConfig(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}
`, name)
}

func testAccPolicySchemaDataSourceConfig_default(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_policy_schema" "test" {
  zone_id = keycard_zone.test.id
}
`, name)
}

func testAccPolicySchemaDataSourceConfig_byVersion(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_policy_schema" "default" {
  zone_id = keycard_zone.test.id
}

data "keycard_policy_schema" "by_version" {
  zone_id = keycard_zone.test.id
  version = data.keycard_policy_schema.default.version
}
`, name)
}

func testAccPolicySchemaDataSourceConfig_versionNotFound(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_policy_schema" "test" {
  zone_id = keycard_zone.test.id
  version = "0000-00-00"
}
`, name)
}
