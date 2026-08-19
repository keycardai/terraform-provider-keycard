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
	namespace := acctest.RandomWithPrefix("ns")
	serviceAccount := acctest.RandomWithPrefix("sa")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a workload identity resource and fetch it with the data source
			{
				Config: testAccApplicationWorkloadIdentityDataSourceConfig_basic(rName, namespace, serviceAccount),
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

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a workload identity without subject and fetch it
			{
				Config: testAccApplicationWorkloadIdentityDataSourceConfig_noSubject(rName),
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
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Attempt to fetch a workload identity that doesn't exist
			{
				Config:      testAccApplicationWorkloadIdentityDataSourceConfig_notFound(),
				ExpectError: regexp.MustCompile("Application Workload Identity Not Found"),
			},
		},
	})
}

func testAccApplicationWorkloadIdentityDataSourceConfig_basic(appName, namespace, serviceAccount string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name        = "k8s-provider-%[1]s"

  oauth2 = {
    issuer = "https://kubernetes.default.svc.cluster.local"
  }
  zone_id     = data.keycard_organization.test.zone_id
}

resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_workload_identity" "test" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  provider_id    = keycard_provider.test.id
  subject        = "system:serviceaccount:%[2]s:%[3]s"
}

data "keycard_application_workload_identity" "test" {
  zone_id = keycard_application_workload_identity.test.zone_id
  id      = keycard_application_workload_identity.test.id
}
`, appName, namespace, serviceAccount)
}

func testAccApplicationWorkloadIdentityDataSourceConfig_noSubject(appName string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name        = "k8s-provider-%[1]s"

  oauth2 = {
    issuer = "https://kubernetes.default.svc.cluster.local"
  }
  zone_id     = data.keycard_organization.test.zone_id
}

resource "keycard_application" "test" {
  name       = %[1]q
  identifier = "https://%[1]s.example.com"
  zone_id    = data.keycard_organization.test.zone_id
}

resource "keycard_application_workload_identity" "test" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  provider_id    = keycard_provider.test.id
}

data "keycard_application_workload_identity" "test" {
  zone_id = keycard_application_workload_identity.test.zone_id
  id      = keycard_application_workload_identity.test.id
}
`, appName)
}

func testAccApplicationWorkloadIdentityDataSourceConfig_notFound() string {
	return testAccOrgZone + `
data "keycard_application_workload_identity" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "non-existent-workload-identity-id-12345"
}
`
}
