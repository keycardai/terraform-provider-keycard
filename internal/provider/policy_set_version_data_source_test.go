package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

func TestAccPolicySetVersionDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySetVersionDataSourceConfig(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_policy_set_version.test", "id", "keycard_policy_set_version.test", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_set_version.test", "policy_set_id", "keycard_policy_set.test", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_set_version.test", "manifest_sha", "keycard_policy_set_version.test", "manifest_sha"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_set_version.test", "version", "keycard_policy_set_version.test", "version"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_set_version.test", "schema_version", "keycard_policy_set_version.test", "schema_version"),
					resource.TestCheckResourceAttr("data.keycard_policy_set_version.test", "manifest.#", "1"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_set_version.test", "manifest.0.policy_id", "keycard_policy.test", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_set_version.test", "manifest.0.policy_version_id", "keycard_policy_version.test", "id"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_set_version.test", "manifest.0.sha"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_set_version.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_set_version.test", "created_by"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_set_version.test", "owner_type"),
					resource.TestCheckNoResourceAttr("data.keycard_policy_set_version.test", "archived_at"),
					resource.TestCheckNoResourceAttr("data.keycard_policy_set_version.test", "archived_by"),
				),
			},
		},
	})
}

// Pins the svc-pdp contract that an archived set version still reads as 200
// with archived_at set: the version is archived out-of-band, then a refresh
// re-reads the data source and must surface the archived timestamps instead of
// erroring.
func TestAccPolicySetVersionDataSource_archived(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	var zoneID, policySetID, versionID string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySetVersionDataSourceConfig_archived(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCaptureResourceAttr("keycard_zone.test", "id", &zoneID),
					testAccCaptureResourceAttr("keycard_policy_set.test", "id", &policySetID),
					testAccCaptureResourceAttr("keycard_policy_set_version.test", "id", &versionID),
					resource.TestCheckNoResourceAttr("data.keycard_policy_set_version.test", "archived_at"),
				),
			},
			// The out-of-band archive drift-removes the managed version resource,
			// which this apply recreates. The data source's id is pinned to the
			// archived version through terraform_data (ignore_changes keeps it
			// from re-pointing at the replacement), so its re-read must succeed
			// and report the archive.
			{
				PreConfig: testAccArchivePolicySetVersionOutOfBand(t, &zoneID, &policySetID, &versionID),
				Config:    testAccPolicySetVersionDataSourceConfig_archived(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.keycard_policy_set_version.test", "archived_at"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_set_version.test", "archived_by"),
				),
			},
		},
	})
}

func TestAccPolicySetVersionDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesShortRetry,
		Steps: []resource.TestStep{
			// Zone-only first step: creating other resources here risks a
			// non-empty refresh plan when the short-retry provider reads them
			// while the zone is still bootstrapping.
			{
				Config: testAccZoneOnlyConfig(zoneName),
			},
			{
				PreConfig:   waitForZoneBootstrap,
				Config:      testAccPolicySetVersionDataSourceConfig_notFound(zoneName, rName),
				ExpectError: regexp.MustCompile(`Policy Set Version Not Found`),
			},
		},
	})
}

// testAccArchivePolicySetVersionOutOfBand returns a PreConfig that archives a
// policy set version directly against the API, simulating an out-of-band
// archive. Pointers are dereferenced at call time so captured attribute values
// are resolved.
func testAccArchivePolicySetVersionOutOfBand(t *testing.T, zoneID, policySetID, versionID *string) func() {
	return func() {
		t.Helper()

		ctx := context.Background()
		c := testAccOutOfBandClient(t)
		// The version can lag a replica; retry transient 404s.
		resp, err := callWithRetry(ctx, func() (*client.ArchivePolicySetVersionResponse, error) {
			return c.ArchivePolicySetVersionWithResponse(ctx, *zoneID, *policySetID, *versionID)
		}, retryOnNotFound)
		if err != nil {
			t.Fatalf("out-of-band archive failed: %s", err)
		}
		if resp.StatusCode() != 200 {
			t.Fatalf("out-of-band archive failed, got status %d: %s", resp.StatusCode(), string(resp.Body))
		}
	}
}

// The fixture deliberately does not activate the set version: an active zone
// binding would block archiving the bound resources at teardown.
func testAccPolicySetVersionDataSourceConfig(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

data "keycard_policy_schema" "default" {
  zone_id = keycard_zone.test.id
}

resource "keycard_policy_version" "test" {
  zone_id        = keycard_zone.test.id
  policy_id      = keycard_policy.test.id
  schema_version = data.keycard_policy_schema.default.version
  cedar          = "permit (principal, action, resource);\n"
}

resource "keycard_policy_set" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

resource "keycard_policy_set_version" "test" {
  zone_id        = keycard_zone.test.id
  policy_set_id  = keycard_policy_set.test.id
  schema_version = data.keycard_policy_schema.default.version

  manifest = [
    {
      policy_id         = keycard_policy.test.id
      policy_version_id = keycard_policy_version.test.id
    }
  ]
}

data "keycard_policy_set_version" "test" {
  zone_id       = keycard_zone.test.id
  policy_set_id = keycard_policy_set.test.id
  id            = keycard_policy_set_version.test.id
}
`, zoneName, name)
}

func testAccPolicySetVersionDataSourceConfig_archived(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

data "keycard_policy_schema" "default" {
  zone_id = keycard_zone.test.id
}

resource "keycard_policy_version" "test" {
  zone_id        = keycard_zone.test.id
  policy_id      = keycard_policy.test.id
  schema_version = data.keycard_policy_schema.default.version
  cedar          = "permit (principal, action, resource);\n"
}

resource "keycard_policy_set" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

resource "keycard_policy_set_version" "test" {
  zone_id        = keycard_zone.test.id
  policy_set_id  = keycard_policy_set.test.id
  schema_version = data.keycard_policy_schema.default.version

  manifest = [
    {
      policy_id         = keycard_policy.test.id
      policy_version_id = keycard_policy_version.test.id
    }
  ]
}

# Pins the version ID so the data source keeps reading the original (archived)
# version even after the drift-removed version resource is recreated.
resource "terraform_data" "version_id" {
  input = keycard_policy_set_version.test.id

  lifecycle {
    ignore_changes = [input]
  }
}

data "keycard_policy_set_version" "test" {
  zone_id       = keycard_zone.test.id
  policy_set_id = keycard_policy_set.test.id
  id            = terraform_data.version_id.output
}
`, zoneName, name)
}

func testAccPolicySetVersionDataSourceConfig_notFound(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy_set" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

data "keycard_policy_set_version" "test" {
  zone_id       = keycard_zone.test.id
  policy_set_id = keycard_policy_set.test.id
  id            = "does-not-exist-version-id"
}
`, zoneName, name)
}
