package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Note: the "cannot archive a currently-bound version" 400 path is not covered
// here because reproducing it requires the not-yet-implemented
// keycard_policy_set_activation resource to bind the version first. Delete only
// exercises the unbound (happy) archive path.

func TestAccPolicySetVersionResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Bootstrap the zone, policy, and policy set first.
			{
				Config: testAccPolicySetVersionConfig(zoneName, rName, "permit (principal, action, resource);\n", false),
			},
			// Create and Read the set version once the zone has propagated.
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySetVersionConfig(zoneName, rName, "permit (principal, action, resource);\n", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy_set_version.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_policy_set_version.test", "version"),
					resource.TestCheckResourceAttrSet("keycard_policy_set_version.test", "manifest_sha"),
					resource.TestCheckResourceAttrSet("keycard_policy_set_version.test", "created_at"),
					resource.TestCheckResourceAttr("keycard_policy_set_version.test", "active", "false"),
					resource.TestCheckResourceAttr("keycard_policy_set_version.test", "owner_type", "customer"),
					resource.TestCheckResourceAttr("keycard_policy_set_version.test", "manifest.#", "1"),
					resource.TestCheckResourceAttrSet("keycard_policy_set_version.test", "manifest.0.sha"),
					resource.TestCheckResourceAttrPair("keycard_policy_set_version.test", "policy_set_id", "keycard_policy_set.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_policy_set_version.test", "manifest.0.policy_id", "keycard_policy.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_policy_set_version.test", "manifest.0.policy_version_id", "keycard_policy_version.test", "id"),
				),
			},
			// Import round-trips the full manifest (including computed shas).
			{
				ResourceName:      "keycard_policy_set_version.test",
				ImportState:       true,
				ImportStateIdFunc: testAccPolicySetVersionImportStateIdFunc("keycard_policy_set_version.test"),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPolicySetVersionResource_manifestChangeForcesNew(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetVersionConfig(zoneName, rName, "permit (principal, action, resource);\n", true),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySetVersionConfig(zoneName, rName, "permit (principal, action, resource);\n", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy_set_version.test", "manifest_sha"),
				),
			},
			// Changing the manifest (via a new policy version) must replace the
			// set version, publishing a new one before archiving the old.
			{
				Config: testAccPolicySetVersionConfig(zoneName, rName, "forbid (principal, action, resource);\n", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("keycard_policy_set_version.test", plancheck.ResourceActionCreateBeforeDestroy),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy_set_version.test", "manifest_sha"),
				),
			},
		},
	})
}

// Helper to generate the import ID in format zone_id/policy_set_id/version_id.
func testAccPolicySetVersionImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		policySetID := rs.Primary.Attributes["policy_set_id"]
		id := rs.Primary.ID

		if zoneID == "" || policySetID == "" || id == "" {
			return "", fmt.Errorf("zone_id, policy_set_id or id is empty")
		}

		return fmt.Sprintf("%s/%s/%s", zoneID, policySetID, id), nil
	}
}

func testAccPolicySetVersionConfig(zoneName, name, cedar string, createBeforeDestroy bool) string {
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

  lifecycle {
    create_before_destroy = true
  }
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
%[4]s}
`, zoneName, name, cedar, lifecycle)
}
