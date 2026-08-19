package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicySchemaDataSource_default(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Fetch the zone's default schema without specifying a version
			{
				Config: testAccPolicySchemaDataSourceConfig_default(),
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
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Look up the default schema, then fetch the same schema by version
			{
				Config: testAccPolicySchemaDataSourceConfig_byVersion(),
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
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesShortRetry,
		Steps: []resource.TestStep{
			// Attempt to fetch a schema version that doesn't exist
			{
				Config:      testAccPolicySchemaDataSourceConfig_versionNotFound(),
				ExpectError: regexp.MustCompile("Policy Schema Not Found"),
			},
		},
	})
}

func testAccPolicySchemaDataSourceConfig_default() string {
	return testAccOrgZone + `
data "keycard_policy_schema" "test" {
  zone_id = data.keycard_organization.test.zone_id
}
`
}

func testAccPolicySchemaDataSourceConfig_byVersion() string {
	return testAccOrgZone + `
data "keycard_policy_schema" "default" {
  zone_id = data.keycard_organization.test.zone_id
}

data "keycard_policy_schema" "by_version" {
  zone_id = data.keycard_organization.test.zone_id
  version = data.keycard_policy_schema.default.version
}
`
}

func testAccPolicySchemaDataSourceConfig_versionNotFound() string {
	return testAccOrgZone + `
data "keycard_policy_schema" "test" {
  zone_id = data.keycard_organization.test.zone_id
  version = "0000-00-00"
}
`
}
