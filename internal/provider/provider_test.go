package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"keycard": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesShortRetry configures a provider whose
// not-found retry window is short, for tests that exercise a genuine not-found
// and would otherwise wait out the production window. Instance-scoped so it is
// safe to use alongside the parallel default-factory tests.
var testAccProtoV6ProviderFactoriesShortRetry = map[string]func() (tfprotov6.ProviderServer, error){
	"keycard": providerserver.NewProtocol6WithError(&KeycardProvider{
		version:             "test",
		retryWindowOverride: 2 * time.Second,
	}),
}

// testAccOrgZone exposes the organization's built-in zone. Tests that just need
// somewhere to hang resources use it instead of creating a zone: zone creation
// and teardown are slow, and the policy service negatively caches a freshly
// created zone for up to a minute. Tests whose subject is the zone itself, that
// write zone-keyed singletons, or that need a guaranteed-empty zone still create
// their own.
const testAccOrgZone = `
data "keycard_organization" "test" {}
`

// testAccOrgZoneRef is the Terraform address of the testAccOrgZone data source,
// for checks comparing a resource's zone_id against the zone it was created in.
const testAccOrgZoneRef = "data.keycard_organization.test"

// testAccOtherZoneRef is the Terraform address of the zone testAccZone declares
// when asked for a zone other than the organization's.
const testAccOtherZoneRef = "keycard_zone.other"

// testAccZone returns the config declaring the zone a step's resources belong
// in, along with the Terraform expression for that zone's id. Tests that assert
// a resource is replaced when zone_id changes move from the organization zone to
// a created one, so only one zone is created per test.
func testAccZone(other bool, name string) (config, zoneID string) {
	if other {
		return fmt.Sprintf(`
resource "keycard_zone" "other" {
  name = %q
}
`, name), testAccOtherZoneRef + ".id"
	}
	return testAccOrgZone, testAccOrgZoneRef + ".zone_id"
}

// testAccZoneOnlyConfig declares a bare zone, for tests that need a zone of
// their own but nothing in it.
func testAccZoneOnlyConfig(name string) string {
	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}
`, name)
}

// waitForZoneBootstrap gives a newly created zone time to register with the
// policy service, which negatively caches unknown zones for up to a minute. The
// data sources retry through that, but tests run much faster when the first read
// lands after bootstrap instead of poisoning the cache. Tests using
// testAccOrgZone do not need it.
func waitForZoneBootstrap() {
	time.Sleep(5 * time.Second)
}

func testAccPreCheckBasic(t *testing.T) {
	requiredEnvVars := []string{
		"KEYCARD_CLIENT_ID",
		"KEYCARD_CLIENT_SECRET",
		"KEYCARD_ENDPOINT",
	}

	for _, envVar := range requiredEnvVars {
		if v := os.Getenv(envVar); v == "" {
			t.Fatalf("%s must be set for acceptance tests", envVar)
		}
	}
}

func testAccPreCheck(t *testing.T) {
	testAccPreCheckBasic(t)

	requiredEnvVars := []string{
		"KEYCARD_TEST_KMS_KEY_1",
		"KEYCARD_TEST_KMS_KEY_2",
	}

	for _, envVar := range requiredEnvVars {
		if v := os.Getenv(envVar); v == "" {
			t.Fatalf("%s must be set for acceptance tests", envVar)
		}
	}
}
