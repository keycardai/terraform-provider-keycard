package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExternalSyncTokenResource_basic(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	providerName := acctest.RandomWithPrefix("tftest-provider")
	identifier := fmt.Sprintf("https://%s.example.com", providerName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExternalSyncTokenResourceConfig(zoneName, providerName, identifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_external_sync_token.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_external_sync_token.test", "token"),
					resource.TestCheckResourceAttrPair(
						"keycard_external_sync_token.test", "zone_id",
						"keycard_zone.test", "id",
					),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccExternalSyncTokenResourceConfig(zoneName, providerName, identifier string) string {
	return testAccZoneUserIdentityConfigResourceConfig_externalSync(zoneName, providerName, identifier, true) + `
resource "keycard_external_sync_token" "test" {
  zone_id    = keycard_zone.test.id
  depends_on = [keycard_zone_user_identity_config.test]
}
`
}
