package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPolicyVersionResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Bootstrap the zone and policy container first.
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName),
			},
			// Create and Read the version.
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicyVersionConfig(zoneName, rName, "permit (principal, action, resource);\n", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "version"),
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "sha"),
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "created_at"),
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "owner_type"),
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "schema_version"),
					resource.TestCheckResourceAttrPair("keycard_policy_version.test", "policy_id", "keycard_policy.test", "id"),
				),
			},
			// Import. Server-normalized cedar may differ from the configured
			// value, so cedar is excluded from import verification.
			{
				ResourceName:            "keycard_policy_version.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccPolicyVersionImportStateIdFunc("keycard_policy_version.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"cedar"},
			},
		},
	})
}

func TestAccPolicyVersionResource_contentChangeForcesNew(t *testing.T) {
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
				Config:    testAccPolicyVersionConfig(zoneName, rName, "permit (principal, action, resource);\n", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "sha"),
				),
			},
			// Changing cedar must replace the resource (publish a new version).
			{
				Config: testAccPolicyVersionConfig(zoneName, rName, "forbid (principal, action, resource);\n", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("keycard_policy_version.test", plancheck.ResourceActionCreateBeforeDestroy),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "sha"),
				),
			},
		},
	})
}

// Applying non-canonical Cedar formatting must not produce a perpetual diff:
// the configured value is preserved rather than overwritten by the server's
// normalized form. The framework's post-apply plan check enforces this.
func TestAccPolicyVersionResource_normalizationRoundTrip(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	messyCedar := "permit (\n\n    principal,\n        action,\n\n  resource\n);\n\n"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicyVersionConfig(zoneName, rName, messyCedar, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_version.test", "cedar", messyCedar),
				),
			},
		},
	})
}

// TestAccPolicyVersionResource_recreatedWhenArchivedOutOfBand verifies the
// drift path: when a version is archived behind Terraform's back, the next
// refresh observes the archive (404 or 200 with archived_at set), drops it
// from state, and plans to recreate it. Uses the short-retry factory so the
// Read's 404 retry is bounded to seconds rather than the full production window.
func TestAccPolicyVersionResource_recreatedWhenArchivedOutOfBand(t *testing.T) {
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
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicyVersionConfig(zoneName, rName, "permit (principal, action, resource);\n", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy_version.test", "id"),
					testAccCheckPolicyVersionArchivedOutOfBand(t, "keycard_policy_version.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// testAccCheckPolicyVersionArchivedOutOfBand archives the version directly
// through the API, then polls until the archive has propagated so the
// subsequent refresh reliably observes it (either a 404 or a 200 with
// archived_at set).
func testAccCheckPolicyVersionArchivedOutOfBand(t *testing.T, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		zoneID := rs.Primary.Attributes["zone_id"]
		policyID := rs.Primary.Attributes["policy_id"]
		versionID := rs.Primary.ID

		c := testAccAPIClient(t)
		ctx := context.Background()

		archiveResp, err := c.ArchivePolicyVersionWithResponse(ctx, zoneID, policyID, versionID)
		if err != nil {
			return fmt.Errorf("out-of-band archive failed: %w", err)
		}
		if archiveResp.StatusCode() != 200 && archiveResp.StatusCode() != 404 {
			return fmt.Errorf("out-of-band archive returned status %d: %s", archiveResp.StatusCode(), string(archiveResp.Body))
		}

		// Require several consecutive archived observations: reads can hit
		// divergent replicas right after the archive, so a single hit does not
		// mean the provider's next refresh will also see it.
		consecutiveArchived := 0
		for range 30 {
			getResp, err := c.GetPolicyVersionWithResponse(ctx, zoneID, policyID, versionID)
			archived := err == nil && (getResp.StatusCode() == 404 ||
				(getResp.StatusCode() == 200 && getResp.JSON200 != nil && isArchived(getResp.JSON200.ArchivedAt)))
			if archived {
				consecutiveArchived++
				if consecutiveArchived >= 3 {
					return nil
				}
			} else {
				consecutiveArchived = 0
			}
			time.Sleep(time.Second)
		}
		return fmt.Errorf("policy version %s still readable after out-of-band archive", versionID)
	}
}

// Helper to generate the import ID in format zone_id/policy_id/version_id.
func testAccPolicyVersionImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		policyID := rs.Primary.Attributes["policy_id"]
		id := rs.Primary.ID

		if zoneID == "" || policyID == "" || id == "" {
			return "", fmt.Errorf("zone_id, policy_id or id is empty")
		}

		return fmt.Sprintf("%s/%s/%s", zoneID, policyID, id), nil
	}
}

func testAccPolicyVersionConfig(zoneName, policyName, cedar string, createBeforeDestroy bool) string {
	lifecycle := ""
	if createBeforeDestroy {
		lifecycle = "\n  lifecycle {\n    create_before_destroy = true\n  }\n"
	}

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
  cedar          = %[3]q
%[4]s}
`, zoneName, policyName, cedar, lifecycle)
}
