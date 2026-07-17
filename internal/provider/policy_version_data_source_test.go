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

func TestAccPolicyVersionDataSource_basic(t *testing.T) {
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
				Config:    testAccPolicyVersionDataSourceConfig(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.keycard_policy_version.test", "id", "keycard_policy_version.test", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_version.test", "policy_id", "keycard_policy.test", "id"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_version.test", "sha", "keycard_policy_version.test", "sha"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_version.test", "version", "keycard_policy_version.test", "version"),
					resource.TestCheckResourceAttrPair("data.keycard_policy_version.test", "schema_version", "keycard_policy_version.test", "schema_version"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_version.test", "cedar"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_version.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_version.test", "created_by"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_version.test", "owner_type"),
					resource.TestCheckNoResourceAttr("data.keycard_policy_version.test", "archived_at"),
					resource.TestCheckNoResourceAttr("data.keycard_policy_version.test", "archived_by"),
				),
			},
		},
	})
}

// Pins the svc-pdp contract that an archived version still reads as 200 with
// archived_at set: the version is archived out-of-band, then a refresh re-reads
// the data source and must surface the archived timestamps instead of erroring.
func TestAccPolicyVersionDataSource_archived(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	var zoneID, policyID, versionID string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicyVersionDataSourceConfig_archived(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCaptureResourceAttr("keycard_zone.test", "id", &zoneID),
					testAccCaptureResourceAttr("keycard_policy.test", "id", &policyID),
					testAccCaptureResourceAttr("keycard_policy_version.test", "id", &versionID),
					resource.TestCheckNoResourceAttr("data.keycard_policy_version.test", "archived_at"),
				),
			},
			// The out-of-band archive drift-removes the managed version resource,
			// which this apply recreates. The data source's id is pinned to the
			// archived version through terraform_data (ignore_changes keeps it
			// from re-pointing at the replacement), so its re-read must succeed
			// and report the archive.
			{
				PreConfig: testAccArchivePolicyVersionOutOfBand(t, &zoneID, &policyID, &versionID),
				Config:    testAccPolicyVersionDataSourceConfig_archived(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.keycard_policy_version.test", "archived_at"),
					resource.TestCheckResourceAttrSet("data.keycard_policy_version.test", "archived_by"),
				),
			},
		},
	})
}

func TestAccPolicyVersionDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesShortRetry,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName),
			},
			{
				PreConfig:   waitForZoneBootstrap,
				Config:      testAccPolicyVersionDataSourceConfig_notFound(zoneName, rName),
				ExpectError: regexp.MustCompile(`Policy Version Not Found`),
			},
		},
	})
}

// testAccArchivePolicyVersionOutOfBand returns a PreConfig that archives a
// policy version directly against the API, simulating an out-of-band archive.
// Pointers are dereferenced at call time so captured attribute values are
// resolved.
func testAccArchivePolicyVersionOutOfBand(t *testing.T, zoneID, policyID, versionID *string) func() {
	return func() {
		t.Helper()

		ctx := context.Background()
		c := testAccOutOfBandClient(t)
		// The version can lag a replica; retry transient 404s.
		resp, err := callWithRetry(ctx, func() (*client.ArchivePolicyVersionResponse, error) {
			return c.ArchivePolicyVersionWithResponse(ctx, *zoneID, *policyID, *versionID)
		}, retryOnNotFound)
		if err != nil {
			t.Fatalf("out-of-band archive failed: %s", err)
		}
		if resp.StatusCode() != 200 {
			t.Fatalf("out-of-band archive failed, got status %d: %s", resp.StatusCode(), string(resp.Body))
		}
	}
}

func testAccPolicyVersionDataSourceConfig(zoneName, policyName string) string {
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

data "keycard_policy_version" "test" {
  zone_id   = keycard_zone.test.id
  policy_id = keycard_policy.test.id
  id        = keycard_policy_version.test.id
}
`, zoneName, policyName)
}

func testAccPolicyVersionDataSourceConfig_archived(zoneName, policyName string) string {
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

# Pins the version ID so the data source keeps reading the original (archived)
# version even after the drift-removed version resource is recreated.
resource "terraform_data" "version_id" {
  input = keycard_policy_version.test.id

  lifecycle {
    ignore_changes = [input]
  }
}

data "keycard_policy_version" "test" {
  zone_id   = keycard_zone.test.id
  policy_id = keycard_policy.test.id
  id        = terraform_data.version_id.output
}
`, zoneName, policyName)
}

func testAccPolicyVersionDataSourceConfig_notFound(zoneName, policyName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

data "keycard_policy_version" "test" {
  zone_id   = keycard_zone.test.id
  policy_id = keycard_policy.test.id
  id        = "does-not-exist-version-id"
}
`, zoneName, policyName)
}
