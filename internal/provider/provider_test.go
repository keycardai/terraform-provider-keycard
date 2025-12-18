package provider

import (
	"os"
	"testing"

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
