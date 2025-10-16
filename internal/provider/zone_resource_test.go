// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccZoneResourceConfig_basic(rName + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName+"-updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccZoneResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with description
			{
				Config: testAccZoneResourceConfig_withDescription(rName, "Test description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "description", "Test description"),
				),
			},
			// Update description
			{
				Config: testAccZoneResourceConfig_withDescription(rName, "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "description", "Updated description"),
				),
			},
			// Remove description
			{
				Config: testAccZoneResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "description", ""),
				),
			},
		},
	})
}

func TestAccZoneResource_withOAuth2(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with OAuth2 settings
			{
				Config: testAccZoneResourceConfig_withOAuth2(rName, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "true"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "true"),
				),
			},
			// Update OAuth2 settings
			{
				Config: testAccZoneResourceConfig_withOAuth2(rName, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "false"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "false"),
				),
			},
		},
	})
}

func TestAccZoneResource_complete(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all fields
			{
				Config: testAccZoneResourceConfig_complete(rName, "Complete zone", true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_zone.test", "name", rName),
					resource.TestCheckResourceAttr("keycard_zone.test", "description", "Complete zone"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.pkce_required", "true"),
					resource.TestCheckResourceAttr("keycard_zone.test", "oauth2.dcr_enabled", "false"),
					resource.TestCheckResourceAttrSet("keycard_zone.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccZoneResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}
`, name)
}

func testAccZoneResourceConfig_withDescription(name, description string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name        = %[1]q
  description = %[2]q
}
`, name, description)
}

func testAccZoneResourceConfig_withOAuth2(name string, pkceRequired, dcrEnabled bool) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q

  oauth2 {
    pkce_required = %[2]t
    dcr_enabled   = %[3]t
  }
}
`, name, pkceRequired, dcrEnabled)
}

func testAccZoneResourceConfig_complete(name, description string, pkceRequired, dcrEnabled bool) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name        = %[1]q
  description = %[2]q

  oauth2 {
    pkce_required = %[3]t
    dcr_enabled   = %[4]t
  }
}
`, name, description, pkceRequired, dcrEnabled)
}
