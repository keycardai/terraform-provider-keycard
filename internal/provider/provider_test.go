// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"keycard": providerserver.NewProtocol6WithError(New("test")()),
}

// TestAccProvider_MissingConfiguration tests that the provider fails when required configuration is missing.
func TestAccProvider_MissingConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      `provider "keycard" {}`,
				ExpectError: regexp.MustCompile("Missing Organization ID"),
			},
		},
	})
}

// TestAccProvider_EnvironmentVariables tests that environment variables are used as fallback.
func TestAccProvider_EnvironmentVariables(t *testing.T) {
	// Set test environment variables
	t.Setenv("KEYCARD_ORGANIZATION_ID", "test-org-id")
	t.Setenv("KEYCARD_CLIENT_ID", "test-client-id")
	t.Setenv("KEYCARD_CLIENT_SECRET", "test-client-secret")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "keycard" {}`,
				// Provider should configure successfully with environment variables
				// (test will fail if provider configuration fails)
			},
		},
	})
}

// TestAccProvider_ExplicitConfiguration tests that explicit configuration works.
func TestAccProvider_ExplicitConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "keycard" {
					organization_id = "test-org-id"
					client_id       = "test-client-id"
					client_secret   = "test-client-secret"
					endpoint        = "https://api.keycard-test.ai"
				}`,
				// Provider should configure successfully with explicit configuration
				// (test will fail if provider configuration fails)
			},
		},
	})
}
