package provider

import (
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
		notFoundRetryWindow: 2 * time.Second,
	}),
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
