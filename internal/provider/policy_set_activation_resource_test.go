package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/oapi-codegen/nullable"

	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

func TestAccPolicySetActivationResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	var zoneID, schemaVersion string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Bootstrap the zone and policy chain without the activation.
			{
				Config: testAccPolicySetActivationConfig(zoneName, rName, 0),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySetActivationConfig(zoneName, rName, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_policy_set_activation.test", "id", "keycard_zone.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_policy_set_activation.test", "zone_id", "keycard_zone.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_policy_set_activation.test", "policy_set_id", "keycard_policy_set.test", "id"),
					resource.TestCheckResourceAttrPair("keycard_policy_set_activation.test", "policy_set_version_id", "keycard_policy_set_version.v1", "id"),
					testAccCaptureResourceAttr("keycard_zone.test", "id", &zoneID),
					testAccCaptureResourceAttr("data.keycard_policy_schema.default", "version", &schemaVersion),
				),
			},
			// The activated version reports active once refreshed.
			{
				Config: testAccPolicySetActivationConfig(zoneName, rName, 1),
				Check:  resource.TestCheckResourceAttr("keycard_policy_set_version.v1", "active", "true"),
			},
			{
				ResourceName:      "keycard_policy_set_activation.test",
				ImportState:       true,
				ImportStateIdFunc: testAccPolicySetActivationImportStateIdFunc("keycard_policy_set_activation.test"),
				ImportStateVerify: true,
			},
			// Free the zone's binding so teardown can archive the managed
			// resources (see testAccReleaseZoneBinding).
			{
				PreConfig: testAccReleaseZoneBinding(t, &zoneID, &schemaVersion),
				Config:    testAccPolicySetActivationConfig(zoneName, rName, 0),
			},
		},
	})
}

func TestAccPolicySetActivationResource_rollForwardAndBack(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	zoneName := acctest.RandomWithPrefix("tftest-zone")

	var zoneID, schemaVersion, activationID string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicySetActivationConfig(zoneName, rName, 0),
			},
			{
				PreConfig: waitForZoneBootstrap,
				Config:    testAccPolicySetActivationConfig(zoneName, rName, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_policy_set_activation.test", "policy_set_version_id", "keycard_policy_set_version.v1", "id"),
					testAccCaptureResourceAttr("keycard_policy_set_activation.test", "id", &activationID),
					testAccCaptureResourceAttr("keycard_zone.test", "id", &zoneID),
					testAccCaptureResourceAttr("data.keycard_policy_schema.default", "version", &schemaVersion),
				),
			},
			// Roll forward to v2: in-place update, identity unchanged.
			{
				Config: testAccPolicySetActivationConfig(zoneName, rName, 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("keycard_policy_set_activation.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_policy_set_activation.test", "policy_set_version_id", "keycard_policy_set_version.v2", "id"),
					resource.TestCheckResourceAttrPtr("keycard_policy_set_activation.test", "id", &activationID),
				),
			},
			// Roll back to v1: the server has no monotonicity guard.
			{
				Config: testAccPolicySetActivationConfig(zoneName, rName, 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("keycard_policy_set_activation.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("keycard_policy_set_activation.test", "policy_set_version_id", "keycard_policy_set_version.v1", "id"),
					resource.TestCheckResourceAttrPtr("keycard_policy_set_activation.test", "id", &activationID),
				),
			},
			{
				PreConfig: testAccReleaseZoneBinding(t, &zoneID, &schemaVersion),
				Config:    testAccPolicySetActivationConfig(zoneName, rName, 0),
			},
		},
	})
}

// Helper to generate the import ID in format zone_id/policy_set_id.
func testAccPolicySetActivationImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		zoneID := rs.Primary.Attributes["zone_id"]
		policySetID := rs.Primary.Attributes["policy_set_id"]

		if zoneID == "" || policySetID == "" {
			return "", fmt.Errorf("zone_id or policy_set_id is empty")
		}

		return fmt.Sprintf("%s/%s", zoneID, policySetID), nil
	}
}

// testAccOutOfBandClient builds an API client outside the provider, for
// simulating out-of-band changes and freeing bindings before teardown.
func testAccOutOfBandClient(t *testing.T) *client.ClientWithResponses {
	t.Helper()

	c, err := client.NewAPIClient(context.Background(), client.Config{
		ClientID:     os.Getenv("KEYCARD_CLIENT_ID"),
		ClientSecret: os.Getenv("KEYCARD_CLIENT_SECRET"),
		Endpoint:     os.Getenv("KEYCARD_ENDPOINT"),
	})
	if err != nil {
		t.Fatalf("failed to create out-of-band API client: %s", err)
	}
	return c
}

func activateOutOfBand(t *testing.T, c *client.ClientWithResponses, zoneID, policySetID, versionID string) {
	t.Helper()

	ctx := context.Background()
	// The zone/set/version can lag a replica; retry transient 404s.
	resp, err := callWithRetry(ctx, func() (*client.UpdatePolicySetVersionResponse, error) {
		return c.UpdatePolicySetVersionWithResponse(ctx, zoneID, policySetID, versionID, client.UpdatePolicySetVersionRequest{Active: true})
	}, retryOnNotFound)
	if err != nil {
		t.Fatalf("out-of-band activation failed: %s", err)
	}
	if resp.StatusCode() != 200 {
		t.Fatalf("out-of-band activation failed, got status %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

// testAccReleaseZoneBinding frees the zone's single active binding before
// framework teardown. There is no deactivation API, and while a binding is
// active the server refuses to archive the bound set version, the policy
// versions it references, and the policy set — so destroy would fail. The
// binding is keyed by zone, so activating a sacrificial policy set version
// (created out-of-band, never in Terraform state) releases every managed
// resource for archiving. The sacrificial policy/set/versions stay behind in
// the test zone; they cannot be archived while bound and are abandoned with it.
func testAccReleaseZoneBinding(t *testing.T, zoneID, schemaVersion *string) func() {
	return func() {
		t.Helper()

		ctx := context.Background()
		c := testAccOutOfBandClient(t)
		name := acctest.RandomWithPrefix("tftest-sacrificial")

		// These run out-of-band against the raw client, so unlike the provider
		// they get no automatic retry. The PDP endpoints negatively cache an
		// unknown zone for up to a minute, so a fresh zone can 404 here; retry
		// transient 404s the way the provider does.
		policyResp, err := callWithRetry(ctx, func() (*client.CreatePolicyResponse, error) {
			return c.CreatePolicyWithResponse(ctx, *zoneID, client.CreatePolicyRequest{Name: name})
		}, retryOnNotFound)
		if err != nil {
			t.Fatalf("failed to create sacrificial policy: %s", err)
		}
		if policyResp.JSON201 == nil {
			t.Fatalf("failed to create sacrificial policy, got status %d: %s", policyResp.StatusCode(), string(policyResp.Body))
		}

		policyVersionResp, err := callWithRetry(ctx, func() (*client.CreatePolicyVersionResponse, error) {
			return c.CreatePolicyVersionWithResponse(ctx, *zoneID, policyResp.JSON201.Id, client.CreatePolicyVersionRequest{
				CedarRaw:      nullable.NewNullableWithValue("permit (principal, action, resource);\n"),
				SchemaVersion: *schemaVersion,
			})
		}, retryOnNotFound)
		if err != nil {
			t.Fatalf("failed to create sacrificial policy version: %s", err)
		}
		if policyVersionResp.JSON201 == nil {
			t.Fatalf("failed to create sacrificial policy version, got status %d: %s", policyVersionResp.StatusCode(), string(policyVersionResp.Body))
		}

		setResp, err := callWithRetry(ctx, func() (*client.CreatePolicySetResponse, error) {
			return c.CreatePolicySetWithResponse(ctx, *zoneID, client.CreatePolicySetRequest{Name: name})
		}, retryOnNotFound)
		if err != nil {
			t.Fatalf("failed to create sacrificial policy set: %s", err)
		}
		if setResp.JSON201 == nil {
			t.Fatalf("failed to create sacrificial policy set, got status %d: %s", setResp.StatusCode(), string(setResp.Body))
		}

		setVersionResp, err := callWithRetry(ctx, func() (*client.CreatePolicySetVersionResponse, error) {
			return c.CreatePolicySetVersionWithResponse(ctx, *zoneID, setResp.JSON201.Id, client.CreatePolicySetVersionRequest{
				Manifest: client.PolicySetManifest{Entries: []client.PolicySetManifestEntry{{
					PolicyId:        policyResp.JSON201.Id,
					PolicyVersionId: policyVersionResp.JSON201.Id,
				}}},
				SchemaVersion: *schemaVersion,
			})
		}, retryOnNotFound)
		if err != nil {
			t.Fatalf("failed to create sacrificial policy set version: %s", err)
		}
		if setVersionResp.JSON201 == nil {
			t.Fatalf("failed to create sacrificial policy set version, got status %d: %s", setVersionResp.StatusCode(), string(setVersionResp.Body))
		}

		activateOutOfBand(t, c, *zoneID, setResp.JSON201.Id, setVersionResp.JSON201.Id)
	}
}

func testAccPolicySetActivationConfig(zoneName, name string, activeVersion int) string {
	activation := ""
	if activeVersion > 0 {
		activation = fmt.Sprintf(`
resource "keycard_policy_set_activation" "test" {
  zone_id               = keycard_zone.test.id
  policy_set_id         = keycard_policy_set.test.id
  policy_set_version_id = keycard_policy_set_version.v%d.id
}
`, activeVersion)
	}

	return fmt.Sprintf(`
resource "keycard_zone" "test" {
  name = %[1]q
}

resource "keycard_policy" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

data "keycard_policy_schema" "default" {
  zone_id = keycard_zone.test.id
}

resource "keycard_policy_version" "v1" {
  zone_id        = keycard_zone.test.id
  policy_id      = keycard_policy.test.id
  schema_version = data.keycard_policy_schema.default.version
  cedar          = "permit (principal, action, resource);\n"

  lifecycle {
    create_before_destroy = true
  }
}

resource "keycard_policy_version" "v2" {
  zone_id        = keycard_zone.test.id
  policy_id      = keycard_policy.test.id
  schema_version = data.keycard_policy_schema.default.version
  cedar          = "forbid (principal, action, resource);\n"

  # Serialized: concurrent version creates against the same policy race
  # server-side.
  depends_on = [keycard_policy_version.v1]

  lifecycle {
    create_before_destroy = true
  }
}

resource "keycard_policy_set" "test" {
  name    = %[2]q
  zone_id = keycard_zone.test.id
}

resource "keycard_policy_set_version" "v1" {
  zone_id        = keycard_zone.test.id
  policy_set_id  = keycard_policy_set.test.id
  schema_version = data.keycard_policy_schema.default.version

  manifest = [
    {
      policy_id         = keycard_policy.test.id
      policy_version_id = keycard_policy_version.v1.id
    }
  ]

  lifecycle {
    create_before_destroy = true
  }
}

resource "keycard_policy_set_version" "v2" {
  zone_id        = keycard_zone.test.id
  policy_set_id  = keycard_policy_set.test.id
  schema_version = data.keycard_policy_schema.default.version

  manifest = [
    {
      policy_id         = keycard_policy.test.id
      policy_version_id = keycard_policy_version.v2.id
    }
  ]

  depends_on = [keycard_policy_set_version.v1]

  lifecycle {
    create_before_destroy = true
  }
}
%[3]s`, zoneName, name, activation)
}
