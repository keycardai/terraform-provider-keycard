package provider

import (
	"fmt"
	"regexp"
	"testing"

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
