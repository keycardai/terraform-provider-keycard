package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationWorkloadIdentityResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	namespace := acctest.RandomWithPrefix("ns")
	serviceAccount := acctest.RandomWithPrefix("sa")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_kubernetes(rName, namespace, serviceAccount),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "zone_id"),
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "application_id"),
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "provider_id"),
					resource.TestCheckResourceAttr(
						"keycard_application_workload_identity.test",
						"subject",
						fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceAccount),
					),
					// Verify relationships
					resource.TestCheckResourceAttrPair(
						"keycard_application_workload_identity.test", "zone_id",
						testAccOrgZoneRef, "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_application_workload_identity.test", "application_id",
						"keycard_application.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"keycard_application_workload_identity.test", "provider_id",
						"keycard_provider.test", "id",
					),
				),
			},
			// ImportState testing
			{
				ResourceName:      "keycard_application_workload_identity.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["keycard_application_workload_identity.test"]
					if !ok {
						return "", fmt.Errorf("Not found: keycard_application_workload_identity.test")
					}
					zoneID := rs.Primary.Attributes["zone_id"]
					id := rs.Primary.ID
					return fmt.Sprintf("zones/%s/application-credentials/%s", zoneID, id), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccApplicationWorkloadIdentityResource_updateSubject(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	namespace1 := acctest.RandomWithPrefix("ns1")
	serviceAccount1 := acctest.RandomWithPrefix("sa1")
	namespace2 := acctest.RandomWithPrefix("ns2")
	serviceAccount2 := acctest.RandomWithPrefix("sa2")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first subject
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_kubernetes(rName, namespace1, serviceAccount1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckResourceAttr(
						"keycard_application_workload_identity.test",
						"subject",
						fmt.Sprintf("system:serviceaccount:%s:%s", namespace1, serviceAccount1),
					),
				),
			},
			// Update subject (should NOT force replacement)
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_kubernetes(rName, namespace2, serviceAccount2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckResourceAttr(
						"keycard_application_workload_identity.test",
						"subject",
						fmt.Sprintf("system:serviceaccount:%s:%s", namespace2, serviceAccount2),
					),
				),
			},
		},
	})
}

func TestAccApplicationWorkloadIdentityResource_removeSubject(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	namespace := acctest.RandomWithPrefix("ns")
	serviceAccount := acctest.RandomWithPrefix("sa")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with subject
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_kubernetes(rName, namespace, serviceAccount),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckResourceAttr(
						"keycard_application_workload_identity.test",
						"subject",
						fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceAccount),
					),
				),
			},
			// Remove subject (should accept any token from provider)
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_noSubject(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckNoResourceAttr("keycard_application_workload_identity.test", "subject"),
				),
			},
		},
	})
}

func TestAccApplicationWorkloadIdentityResource_githubActions(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	org := acctest.RandomWithPrefix("org")
	repo := acctest.RandomWithPrefix("repo")
	branch := "main"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_github(rName, org, repo, branch),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckResourceAttr(
						"keycard_application_workload_identity.test",
						"subject",
						fmt.Sprintf("repo:%s/%s:ref:refs/heads/%s", org, repo, branch),
					),
				),
			},
		},
	})
}

func TestAccApplicationWorkloadIdentityResource_awsEks(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	namespace := "kube-system"
	serviceAccount := "aws-node"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_kubernetes(rName, namespace, serviceAccount),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckResourceAttr(
						"keycard_application_workload_identity.test",
						"subject",
						fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceAccount),
					),
				),
			},
		},
	})
}

func TestAccApplicationWorkloadIdentityResource_providerChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	namespace := acctest.RandomWithPrefix("ns")
	serviceAccount := acctest.RandomWithPrefix("sa")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with first provider
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_kubernetes(rName, namespace, serviceAccount),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_workload_identity.test", "provider_id",
						"keycard_provider.test", "id",
					),
				),
			},
			// Change provider (should force replacement)
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_withSecondProvider(rName, namespace, serviceAccount),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.test", "id"),
					resource.TestCheckResourceAttrPair(
						"keycard_application_workload_identity.test", "provider_id",
						"keycard_provider.test2", "id",
					),
				),
			},
		},
	})
}

func TestAccApplicationWorkloadIdentityResource_multipleIdentities(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	namespace1 := "production"
	serviceAccount1 := "app-prod"
	namespace2 := "staging"
	serviceAccount2 := "app-staging"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple workload identities for the same application
			{
				Config: testAccApplicationWorkloadIdentityResourceConfig_multiple(rName, namespace1, serviceAccount1, namespace2, serviceAccount2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// First identity
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.prod", "id"),
					resource.TestCheckResourceAttr(
						"keycard_application_workload_identity.prod",
						"subject",
						fmt.Sprintf("system:serviceaccount:%s:%s", namespace1, serviceAccount1),
					),
					// Second identity
					resource.TestCheckResourceAttrSet("keycard_application_workload_identity.staging", "id"),
					resource.TestCheckResourceAttr(
						"keycard_application_workload_identity.staging",
						"subject",
						fmt.Sprintf("system:serviceaccount:%s:%s", namespace2, serviceAccount2),
					),
					// Both should be for the same application
					resource.TestCheckResourceAttrPair(
						"keycard_application_workload_identity.prod", "application_id",
						"keycard_application_workload_identity.staging", "application_id",
					),
				),
			},
		},
	})
}

func TestAccApplicationWorkloadIdentityResource_emptySubjectInvalid(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApplicationWorkloadIdentityResourceConfig_emptySubject(rName),
				ExpectError: regexp.MustCompile(`Attribute subject string length must be at least 1`),
			},
		},
	})
}

// Helper functions for test configurations

func testAccApplicationWorkloadIdentityResourceConfig_kubernetes(appName, namespace, serviceAccount string) string {
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
`, appName, namespace, serviceAccount)
}

func testAccApplicationWorkloadIdentityResourceConfig_noSubject(appName string) string {
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
`, appName)
}

func testAccApplicationWorkloadIdentityResourceConfig_github(appName, org, repo, branch string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name        = "github-provider-%[1]s"

  oauth2 = {
    issuer = "https://token.actions.githubusercontent.com"
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
  subject        = "repo:%[2]s/%[3]s:ref:refs/heads/%[4]s"
}
`, appName, org, repo, branch)
}

func testAccApplicationWorkloadIdentityResourceConfig_withSecondProvider(appName, namespace, serviceAccount string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name        = "k8s-provider-%[1]s"

  oauth2 = {
    issuer = "https://kubernetes.default.svc.cluster.local"
  }
  zone_id     = data.keycard_organization.test.zone_id
}

resource "keycard_provider" "test2" {
  name        = "k8s-provider2-%[1]s"

  oauth2 = {
    issuer = "https://kubernetes2.default.svc.cluster.local"
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
  provider_id    = keycard_provider.test2.id
  subject        = "system:serviceaccount:%[2]s:%[3]s"
}
`, appName, namespace, serviceAccount)
}

func testAccApplicationWorkloadIdentityResourceConfig_multiple(appName, namespace1, serviceAccount1, namespace2, serviceAccount2 string) string {
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

resource "keycard_application_workload_identity" "prod" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  provider_id    = keycard_provider.test.id
  subject        = "system:serviceaccount:%[2]s:%[3]s"
}

resource "keycard_application_workload_identity" "staging" {
  zone_id        = data.keycard_organization.test.zone_id
  application_id = keycard_application.test.id
  provider_id    = keycard_provider.test.id
  subject        = "system:serviceaccount:%[4]s:%[5]s"
}
`, appName, namespace1, serviceAccount1, namespace2, serviceAccount2)
}

func testAccApplicationWorkloadIdentityResourceConfig_emptySubject(appName string) string {
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
  subject        = ""
}
`, appName)
}
