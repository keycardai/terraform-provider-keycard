package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPolicySetResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_policy_set.test", "target_type", "zone"),
					resource.TestCheckResourceAttrSet("keycard_policy_set.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_policy_set.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_policy_set.test", "owner_type"),
					resource.TestCheckResourceAttrSet("keycard_policy_set.test", "created_by"),
					resource.TestCheckResourceAttrSet("keycard_policy_set.test", "etag"),
					resource.TestCheckResourceAttr("keycard_policy_set.test", "active", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_policy_set.test",
				ImportState:       true,
				ImportStateIdFunc: testAccPolicySetImportStateIdFunc("keycard_policy_set.test"),
				ImportStateVerify: true,
				// etag is repopulated on the first post-import Read, not carried
				// through the import itself.
				ImportStateVerifyIgnore: []string{"etag"},
			},
			// Update name and Read testing
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName, rName+"-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttrSet("keycard_policy_set.test", "etag"),
				),
			},
		},
	})
}

func TestAccPolicySetResource_targetTypeUser(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetResourceConfig_targetType(zoneName, rName, "user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_policy_set.test", "target_type", "user"),
				),
			},
		},
	})
}

func TestAccPolicySetResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName1 := acctest.RandomWithPrefix("tftest-zone1")
	zoneName2 := acctest.RandomWithPrefix("tftest-zone2")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in zone 1
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName1, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_policy_set.test", "zone_id", "keycard_zone.test", "id"),
				),
			},
			// Change zone (should force replacement)
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName2, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_policy_set.test", "zone_id", "keycard_zone.test", "id"),
				),
			},
		},
	})
}

func TestAccPolicySetResource_targetTypeChangeForcesReplacement(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetResourceConfig_targetType(zoneName, rName, "zone"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "target_type", "zone"),
				),
			},
			{
				Config: testAccPolicySetResourceConfig_targetType(zoneName, rName, "user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "target_type", "user"),
				),
			},
		},
	})
}

func TestAccPolicySetResource_emptyNameInvalid(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicySetResourceConfig_basic(zoneName, ""),
				ExpectError: regexp.MustCompile(`Attribute name string length must be at least 1`),
			},
		},
	})
}

func TestAccPolicySetResource_invalidTargetType(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicySetResourceConfig_targetType(zoneName, rName, "resource"),
				ExpectError: regexp.MustCompile(`Attribute target_type value must be one of`),
			},
		},
	})
}

// TestAccPolicySetResource_recreatedWhenArchivedOutOfBand verifies the drift
// path: when a policy set is archived behind Terraform's back, the next refresh
// sees a 404, drops it from state, and plans to recreate it. Uses the
// short-retry factory so the Read's 404 retry is bounded to seconds.
func TestAccPolicySetResource_recreatedWhenArchivedOutOfBand(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesShortRetry,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy_set.test", "id"),
					testAccCheckPolicySetArchivedOutOfBand(t, "keycard_policy_set.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// testAccCheckPolicySetArchivedOutOfBand archives the policy set directly
// through the API, then polls until the archive has propagated so the
// subsequent refresh reliably observes the 404.
func testAccCheckPolicySetArchivedOutOfBand(t *testing.T, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		zoneID := rs.Primary.Attributes["zone_id"]
		policySetID := rs.Primary.ID

		c := testAccAPIClient(t)
		ctx := context.Background()

		archiveResp, err := c.ArchivePolicySetWithResponse(ctx, zoneID, policySetID)
		if err != nil {
			return fmt.Errorf("out-of-band archive failed: %w", err)
		}
		if archiveResp.StatusCode() != 200 && archiveResp.StatusCode() != 404 {
			return fmt.Errorf("out-of-band archive returned status %d: %s", archiveResp.StatusCode(), string(archiveResp.Body))
		}

		// Require several consecutive 404s: reads can hit divergent replicas
		// right after the archive, so a single 404 does not mean the provider's
		// next refresh will also see the miss.
		consecutive404 := 0
		for i := 0; i < 30; i++ {
			getResp, err := c.GetPolicySetWithResponse(ctx, zoneID, policySetID)
			if err == nil && getResp.StatusCode() == 404 {
				consecutive404++
				if consecutive404 >= 3 {
					return nil
				}
			} else {
				consecutive404 = 0
			}
			time.Sleep(time.Second)
		}
		return fmt.Errorf("policy set %s still readable after out-of-band archive", policySetID)
	}
}

// Helper function to generate import state ID in format zones/{zone-id}/policy-sets/{policy-set-id}.
func testAccPolicySetImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		id := rs.Primary.ID

		if zoneID == "" || id == "" {
			return "", fmt.Errorf("zone_id or id is empty")
		}

		return fmt.Sprintf("zones/%s/policy-sets/%s", zoneID, id), nil
	}
}

func testAccPolicySetResourceConfig_basic(zoneName, name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy_set" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}
`, zoneName, name)
}

func testAccPolicySetResourceConfig_targetType(zoneName, name, targetType string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy_set" "test" {
  name        = %[2]q
  target_type = %[3]q
  zone_id     = keycard_zone.test.id
}
`, zoneName, name, targetType)
}
