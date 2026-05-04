package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSOLoginURLDataSource_basic(t *testing.T) {
	issuer := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSSOLoginURLDataSourceConfig_basic(issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_sso_login_url.test", "issuer", issuer),
					resource.TestCheckResourceAttr("data.keycard_sso_login_url.test", "target_link_uri", "https://console.keycard.ai"),
					resource.TestCheckResourceAttrSet("data.keycard_sso_login_url.test", "url"),
				),
			},
		},
	})
}

func TestAccSSOLoginURLDataSource_customTargetLinkURI(t *testing.T) {
	issuer := fmt.Sprintf("https://%s.example.com", acctest.RandomWithPrefix("tftest"))
	targetLinkURI := "https://staging.console.keycard.ai"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSSOLoginURLDataSourceConfig_customTargetLinkURI(issuer, targetLinkURI),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_sso_login_url.test", "issuer", issuer),
					resource.TestCheckResourceAttr("data.keycard_sso_login_url.test", "target_link_uri", targetLinkURI),
					resource.TestCheckResourceAttrSet("data.keycard_sso_login_url.test", "url"),
				),
			},
		},
	})
}

func testAccSSOLoginURLDataSourceConfig_basic(issuer string) string {
	return fmt.Sprintf(`
data "keycard_sso_login_url" "test" {
  issuer = %[1]q
}
`, issuer)
}

func testAccSSOLoginURLDataSourceConfig_customTargetLinkURI(issuer, targetLinkURI string) string {
	return fmt.Sprintf(`
data "keycard_sso_login_url" "test" {
  issuer          = %[1]q
  target_link_uri = %[2]q
}
`, issuer, targetLinkURI)
}
