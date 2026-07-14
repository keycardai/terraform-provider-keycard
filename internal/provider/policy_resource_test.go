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

func TestAccPolicyResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy.test", "name", rName),
					resource.TestCheckResourceAttrSet("keycard_policy.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_policy.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_policy.test", "owner_type"),
					resource.TestCheckResourceAttrSet("keycard_policy.test", "created_by"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_policy.test",
				ImportState:       true,
				ImportStateIdFunc: testAccPolicyImportStateIdFunc("keycard_policy.test"),
				ImportStateVerify: true,
			},
			// Update name and Read testing
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName+"-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy.test", "name", rName+"-updated"),
				),
			},
		},
	})
}

func TestAccPolicyResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccPolicyResourceConfig_withDescription(zoneName, rName, "Test policy description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_policy.test", "description", "Test policy description"),
				),
			},
			// Update description
			{
				Config: testAccPolicyResourceConfig_withDescription(zoneName, rName, "Updated policy description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy.test", "description", "Updated policy description"),
				),
			},
		},
	})
}

func TestAccPolicyResource_zoneChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName1 := acctest.RandomWithPrefix("tftest-zone1")
	zoneName2 := acctest.RandomWithPrefix("tftest-zone2")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create in zone 1
			{
				Config: testAccPolicyResourceConfig_basic(zoneName1, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_policy.test", "zone_id", "keycard_zone.test", "id"),
				),
			},
			// Change zone (should force replacement)
			{
				Config: testAccPolicyResourceConfig_basic(zoneName2, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy.test", "name", rName),
					resource.TestCheckResourceAttrPair("keycard_policy.test", "zone_id", "keycard_zone.test", "id"),
				),
			},
		},
	})
}

func TestAccPolicyResource_emptyNameInvalid(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicyResourceConfig_basic(zoneName, ""),
				ExpectError: regexp.MustCompile(`Attribute name string length must be at least 1`),
			},
		},
	})
}

func TestAccPolicyResource_emptyDescriptionInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicyResourceConfig_withDescription(zoneName, rName, ""),
				ExpectError: regexp.MustCompile(`Attribute description string length must be at least 1`),
			},
		},
	})
}

// TestAccPolicyResource_recreatedWhenArchivedOutOfBand verifies the drift path:
// when a policy is archived behind Terraform's back, the next refresh observes
// the archive (404 or 200 with archived_at set), drops it from state, and plans
// to recreate it. Uses the short-retry factory so the Read's 404 retry is
// bounded to seconds rather than the full production window.
func TestAccPolicyResource_recreatedWhenArchivedOutOfBand(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesShortRetry,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_policy.test", "id"),
					testAccCheckPolicyArchivedOutOfBand(t, "keycard_policy.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// testAccCheckPolicyArchivedOutOfBand archives the policy directly through the
// API, then polls until the archive has propagated so the subsequent refresh
// reliably observes it (either a 404 or a 200 with archived_at set).
func testAccCheckPolicyArchivedOutOfBand(t *testing.T, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		zoneID := rs.Primary.Attributes["zone_id"]
		policyID := rs.Primary.ID

		c := testAccAPIClient(t)
		ctx := context.Background()

		archiveResp, err := c.ArchivePolicyWithResponse(ctx, zoneID, policyID)
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
			getResp, err := c.GetPolicyWithResponse(ctx, zoneID, policyID)
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
		return fmt.Errorf("policy %s still readable after out-of-band archive", policyID)
	}
}

// Helper function to generate import state ID in format zones/{zone-id}/policies/{policy-id}.
func testAccPolicyImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
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

		return fmt.Sprintf("zones/%s/policies/%s", zoneID, id), nil
	}
}

func testAccPolicyResourceConfig_basic(zoneName, policyName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}
`, zoneName, policyName)
}

func testAccPolicyResourceConfig_withDescription(zoneName, policyName, description string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "test" {
  name        = %[2]q
  description = %[3]q
  zone_id     = keycard_zone.test.id
}
`, zoneName, policyName, description)
}
