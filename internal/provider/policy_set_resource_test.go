package provider

import (
	"fmt"
	"regexp"
	"testing"

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

// TestAccPolicySetResource_multipleUpdates runs several sequential updates,
// each a distinct apply (refresh -> update). It guards the real-world case
// where a user applies repeatedly: every update must succeed even though the
// ETag captured by the preceding refresh can be stale under eventual
// consistency. The Update path refetches the ETag and retries on a stale-ETag
// 412 so these all converge.
func TestAccPolicySetResource_multipleUpdates(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName),
					resource.TestCheckResourceAttrSet("keycard_policy_set.test", "etag"),
				),
			},
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName, rName+"-v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName+"-v2"),
				),
			},
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName, rName+"-v3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName+"-v3"),
				),
			},
			{
				Config: testAccPolicySetResourceConfig_basic(zoneName, rName+"-v4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName+"-v4"),
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
			// Changing target_type forces replacement. Use a fresh name so the
			// replacement's create does not collide with the archived set, whose
			// name is not freed synchronously on archive.
			{
				Config: testAccPolicySetResourceConfig_targetType(zoneName, rName+"-user", "user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_policy_set.test", "name", rName+"-user"),
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
