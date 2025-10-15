// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"keycard": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the keycard provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"keycard": providerserver.NewProtocol6WithError(New("test")()),
	"echo":    echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

// TestAccProvider_MissingConfiguration tests that the provider fails when required configuration is missing
func TestAccProvider_MissingConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "keycard" {}`,
				ExpectError: regexp.MustCompile("Missing Organization ID"),
			},
		},
	})
}

// TestAccProvider_EnvironmentVariables tests that environment variables are used as fallback
func TestAccProvider_EnvironmentVariables(t *testing.T) {
	// Save original environment variables
	originalOrgID := os.Getenv("KEYCARD_ORGANIZATION_ID")
	originalClientID := os.Getenv("KEYCARD_CLIENT_ID")
	originalClientSecret := os.Getenv("KEYCARD_CLIENT_SECRET")

	// Set test environment variables
	os.Setenv("KEYCARD_ORGANIZATION_ID", "test-org-id")
	os.Setenv("KEYCARD_CLIENT_ID", "test-client-id")
	os.Setenv("KEYCARD_CLIENT_SECRET", "test-client-secret")

	// Restore original environment variables after test
	defer func() {
		os.Setenv("KEYCARD_ORGANIZATION_ID", originalOrgID)
		os.Setenv("KEYCARD_CLIENT_ID", originalClientID)
		os.Setenv("KEYCARD_CLIENT_SECRET", originalClientSecret)
	}()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "keycard" {}

				data "keycard_test" "example" {}`,
				// Provider should configure successfully with environment variables
				// (test will fail if provider configuration fails)
			},
		},
	})
}

// TestAccProvider_ExplicitConfiguration tests that explicit configuration works
func TestAccProvider_ExplicitConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "keycard" {
					organization_id = "test-org-id"
					client_id       = "test-client-id"
					client_secret   = "test-client-secret"
					endpoint        = "https://api.test.keycard.com"
				}

				data "keycard_test" "example" {}`,
				// Provider should configure successfully with explicit configuration
				// (test will fail if provider configuration fails)
			},
		},
	})
}
