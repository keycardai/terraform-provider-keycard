package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneDirectoryDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone and fetch its built-in directory provider
			{
				Config: testAccZoneDirectoryDataSourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify zone_id is set
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone_directory.test", "zone_id",
						"keycard_zone.test", "id",
					),
					// Verify provider_id is returned and not empty
					resource.TestCheckResourceAttrSet("data.keycard_zone_directory.test", "provider_id"),
				),
			},
		},
	})
}

func TestAccZoneDirectoryDataSource_multipleZones(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tftest")
	rName2 := acctest.RandomWithPrefix("tftest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple zones and verify each has its own directory provider
			{
				Config: testAccZoneDirectoryDataSourceConfig_multipleZones(rName1, rName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify first zone
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone_directory.test1", "zone_id",
						"keycard_zone.test1", "id",
					),
					resource.TestCheckResourceAttrSet("data.keycard_zone_directory.test1", "provider_id"),
					// Verify second zone
					resource.TestCheckResourceAttrPair(
						"data.keycard_zone_directory.test2", "zone_id",
						"keycard_zone.test2", "id",
					),
					resource.TestCheckResourceAttrSet("data.keycard_zone_directory.test2", "provider_id"),
				),
			},
		},
	})
}

func TestAccZoneDirectoryDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch a directory provider for a non-existent zone
			{
				Config:      testAccZoneDirectoryDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile("Zone Not Found"),
			},
		},
	})
}

func testAccZoneDirectoryDataSourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_zone_directory" "test" {
  zone_id = keycard_zone.test.id
}
`, name)
}

func testAccZoneDirectoryDataSourceConfig_multipleZones(name1, name2 string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test1" {
  name = %[1]q
}

resource "keycard_zone" "test2" {
  name = %[2]q
}

data "keycard_zone_directory" "test1" {
  zone_id = keycard_zone.test1.id
}

data "keycard_zone_directory" "test2" {
  zone_id = keycard_zone.test2.id
}
`, name1, name2)
}

func testAccZoneDirectoryDataSourceConfig_notFound() string {
	return `
data "keycard_zone_directory" "test" {
  zone_id = "non-existent-zone-id-12345"
}
`
}
