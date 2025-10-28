package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationWorkloadIdentityDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")
	namespace := acctest.RandomWithPrefix("ns")
	serviceAccount := acctest.RandomWithPrefix("sa")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a workload identity resource and fetch it with the data source
			{
				Config: testAccApplicationWorkloadIdentityDataSourceConfig_basic(zoneName, rName, namespace, serviceAccount),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match the resource
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "id",
						"keycard_application_workload_identity.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "zone_id",
						"keycard_application_workload_identity.test", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "application_id",
						"keycard_application_workload_identity.test", "application_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "provider_id",
						"keycard_application_workload_identity.test", "provider_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "subject",
						"keycard_application_workload_identity.test", "subject",
					),
					resource.TestCheckResourceAttr(
						"data.keycard_application_workload_identity.test",
						"subject",
						fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceAccount),
					),
				),
			},
		},
	})
}

func TestAccApplicationWorkloadIdentityDataSource_noSubject(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a workload identity without subject and fetch it
			{
				Config: testAccApplicationWorkloadIdentityDataSourceConfig_noSubject(zoneName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "id",
						"keycard_application_workload_identity.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "zone_id",
						"keycard_application_workload_identity.test", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "application_id",
						"keycard_application_workload_identity.test", "application_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.keycard_application_workload_identity.test", "provider_id",
						"keycard_application_workload_identity.test", "provider_id",
					),
					// Verify subject is not set (empty)
					resource.TestCheckNoResourceAttr("data.keycard_application_workload_identity.test", "subject"),
				),
			},
		},
	})
}

func TestAccApplicationWorkloadIdentityDataSource_notFound(t *testing.T) {
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone but attempt to fetch a workload identity that doesn't exist
			{
				Config:      testAccApplicationWorkloadIdentityDataSourceConfig_notFound(zoneName),
				ExpectError: regexp.MustCompile("Application Workload Identity Not Found"),
			},
		},
	})
}

func testAccApplicationWorkloadIdentityDataSourceConfig_basic(zoneName, appName, namespace, serviceAccount string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name        = "k8s-provider-%[2]s"
  identifier  = "https://kubernetes.default.svc.cluster.local"
  zone_id     = keycard_zone.test.id
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_application_workload_identity" "test" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  provider_id    = keycard_provider.test.id
  subject        = "system:serviceaccount:%[3]s:%[4]s"
}

data "keycard_application_workload_identity" "test" {
  zone_id = keycard_application_workload_identity.test.zone_id
  id      = keycard_application_workload_identity.test.id
}
`, zoneName, appName, namespace, serviceAccount)
}

func testAccApplicationWorkloadIdentityDataSourceConfig_noSubject(zoneName, appName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_provider" "test" {
  name        = "k8s-provider-%[2]s"
  identifier  = "https://kubernetes.default.svc.cluster.local"
  zone_id     = keycard_zone.test.id
}

resource "keycard_application" "test" {
  name       = %[2]q
  identifier = "https://%[2]s.example.com"
  zone_id    = keycard_zone.test.id
}

resource "keycard_application_workload_identity" "test" {
  zone_id        = keycard_zone.test.id
  application_id = keycard_application.test.id
  provider_id    = keycard_provider.test.id
}

data "keycard_application_workload_identity" "test" {
  zone_id = keycard_application_workload_identity.test.zone_id
  id      = keycard_application_workload_identity.test.id
}
`, zoneName, appName)
}

func testAccApplicationWorkloadIdentityDataSourceConfig_notFound(zoneName string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

data "keycard_application_workload_identity" "test" {
  zone_id = keycard_zone.test.id
  id      = "non-existent-workload-identity-id-12345"
}
`, zoneName)
}
