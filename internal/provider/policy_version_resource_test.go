package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/keycardai/terraform-provider-keycard/internal/client"
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
					resource.TestCheckNoResourceAttr("keycard_policy_version.test", "archived_at"),
					resource.TestCheckNoResourceAttr("keycard_policy_version.test", "archived_by"),
				),
			},
			// Import. Server-normalized cedar may differ from the configured
			// value, so cedar is excluded from value verification — but it must
			// still be populated: a null cedar on import (e.g. the API not
			// returning cedar_raw) would otherwise slip through the ignore.
			{
				ResourceName:            "keycard_policy_version.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccPolicyVersionImportStateIdFunc("keycard_policy_version.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"cedar"},
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported instance state, got %d", len(states))
					}
					if states[0].Attributes["cedar"] == "" {
						return fmt.Errorf("imported state has empty cedar; expected it populated from the API's cedar_raw")
					}
					return nil
				},
			},
		},
	})
}

func TestAccPolicyVersionResource_contentChangeForcesNew(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	var originalID, originalSha string

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
					testAccCaptureResourceAttr("keycard_policy_version.test", "id", &originalID),
					testAccCaptureResourceAttr("keycard_policy_version.test", "sha", &originalSha),
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
					testAccCheckResourceAttrDiffers("keycard_policy_version.test", "id", &originalID),
					testAccCheckResourceAttrDiffers("keycard_policy_version.test", "sha", &originalSha),
				),
			},
		},
	})
}

// ACC-931: replacing a policy version while the old one is still referenced by
// the zone's active policy set binding must succeed. The binding may be managed
// in another stack or out of band, so no in-config ordering can roll it forward;
// the deposed version cannot be archived (400 "currently active") and is
// forgotten with a warning instead of failing the apply.
func TestAccPolicyVersionResource_replaceWhileActiveBound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	var zoneID, policyID, v1ID, schemaVersion string

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
					testAccCaptureResourceAttr("keycard_zone.test", "id", &zoneID),
					testAccCaptureResourceAttr("keycard_policy.test", "id", &policyID),
					testAccCaptureResourceAttr("keycard_policy_version.test", "id", &v1ID),
					testAccCaptureResourceAttr("data.keycard_policy_schema.default", "version", &schemaVersion),
				),
			},
			// Bind v1 via an out-of-band policy set + activation (never in state),
			// then change cedar: the create-before-destroy replace must succeed and
			// leave the deposed v1 live server-side. Teardown needs no binding
			// release: the replacement version and the policy are unbound, and the
			// out-of-band set is abandoned with the zone like the sacrificial
			// bindings in testAccReleaseZoneBinding.
			{
				PreConfig: testAccBindPolicyVersionOutOfBand(t, &zoneID, &policyID, &v1ID, &schemaVersion),
				Config:    testAccPolicyVersionConfig(zoneName, rName, "forbid (principal, action, resource);\n", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("keycard_policy_version.test", plancheck.ResourceActionCreateBeforeDestroy),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckResourceAttrDiffers("keycard_policy_version.test", "id", &v1ID),
					testAccCheckPolicyVersionLive(t, &zoneID, &policyID, &v1ID),
				),
			},
		},
	})
}

// testAccBindPolicyVersionOutOfBand returns a PreConfig that creates a policy
// set + set version referencing the captured policy version and activates it,
// all outside Terraform state — simulating a binding managed in another stack.
// Pointers are dereferenced at call time so captured values are resolved.
func testAccBindPolicyVersionOutOfBand(t *testing.T, zoneID, policyID, versionID, schemaVersion *string) func() {
	return func() {
		t.Helper()

		ctx := context.Background()
		c := testAccOutOfBandClient(t)
		name := acctest.RandomWithPrefix("tftest-oob")

		setResp, err := callWithRetry(ctx, func() (*client.CreatePolicySetResponse, error) {
			return c.CreatePolicySetWithResponse(ctx, *zoneID, client.CreatePolicySetRequest{Name: name})
		}, retryOnNotFound)
		if err != nil {
			t.Fatalf("failed to create out-of-band policy set: %s", err)
		}
		if setResp.JSON201 == nil {
			t.Fatalf("failed to create out-of-band policy set, got status %d: %s", setResp.StatusCode(), string(setResp.Body))
		}

		setVersionResp, err := callWithRetry(ctx, func() (*client.CreatePolicySetVersionResponse, error) {
			return c.CreatePolicySetVersionWithResponse(ctx, *zoneID, setResp.JSON201.Id, client.CreatePolicySetVersionRequest{
				Manifest: client.PolicySetManifest{Entries: []client.PolicySetManifestEntry{{
					PolicyId:        *policyID,
					PolicyVersionId: *versionID,
				}}},
				SchemaVersion: *schemaVersion,
			})
		}, retryOnNotFound)
		if err != nil {
			t.Fatalf("failed to create out-of-band policy set version: %s", err)
		}
		if setVersionResp.JSON201 == nil {
			t.Fatalf("failed to create out-of-band policy set version, got status %d: %s", setVersionResp.StatusCode(), string(setVersionResp.Body))
		}

		activateOutOfBand(t, c, *zoneID, setResp.JSON201.Id, setVersionResp.JSON201.Id)
	}
}

// testAccCheckPolicyVersionLive asserts the version still exists un-archived
// server-side, i.e. it was forgotten rather than archived.
func testAccCheckPolicyVersionLive(t *testing.T, zoneID, policyID, versionID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		getResp, err := testAccOutOfBandClient(t).GetPolicyVersionWithResponse(context.Background(), *zoneID, *policyID, *versionID)
		if err != nil {
			return fmt.Errorf("failed to read deposed version %s: %w", *versionID, err)
		}
		if getResp.StatusCode() != 200 || getResp.JSON200 == nil {
			return fmt.Errorf("expected deposed version %s to remain live, got status %d: %s", *versionID, getResp.StatusCode(), string(getResp.Body))
		}
		if isArchived(getResp.JSON200.ArchivedAt) {
			return fmt.Errorf("expected deposed version %s to remain un-archived, but archived_at is set", *versionID)
		}
		return nil
	}
}

// testAccCaptureResourceAttr stores the named attribute's current value in dst
// for comparison in a later step.
func testAccCaptureResourceAttr(resourceName, attr string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		v, ok := rs.Primary.Attributes[attr]
		if !ok || v == "" {
			return fmt.Errorf("%s: attribute %q not set", resourceName, attr)
		}
		*dst = v
		return nil
	}
}

// testAccCheckResourceAttrDiffers fails when the named attribute still equals
// the previously captured value, i.e. the resource was not actually replaced.
func testAccCheckResourceAttrDiffers(resourceName, attr string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		v := rs.Primary.Attributes[attr]
		if *previous == "" {
			return fmt.Errorf("%s: no previously captured %q to compare against", resourceName, attr)
		}
		if v == *previous {
			return fmt.Errorf("%s: attribute %q unchanged after expected replacement (still %q)", resourceName, attr, v)
		}
		return nil
	}
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
