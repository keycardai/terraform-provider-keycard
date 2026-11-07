package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccRSAJWK generates a public RSA signing key as a JWK.
func testAccRSAJWK(t *testing.T, kid string) map[string]any {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %s", err)
	}
	b64 := base64.RawURLEncoding.EncodeToString
	return map[string]any{
		"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
		"n": b64(key.N.Bytes()),
		"e": b64(big.NewInt(int64(key.E)).Bytes()),
	}
}

// testAccECJWK generates a public EC P-256 signing key as a JWK.
func testAccECJWK(t *testing.T, kid string) map[string]any {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating EC key: %s", err)
	}
	pub, err := key.PublicKey.ECDH()
	if err != nil {
		t.Fatalf("converting EC key: %s", err)
	}
	// Uncompressed point: 0x04 || X || Y, 32 bytes each for P-256.
	point := pub.Bytes()
	b64 := base64.RawURLEncoding.EncodeToString
	return map[string]any{
		"kty": "EC", "kid": kid, "use": "sig", "alg": "ES256", "crv": "P-256",
		"x": b64(point[1:33]),
		"y": b64(point[33:]),
	}
}

// testAccJWKS returns a JWK Set in compact JSON with sorted members, the form
// both jsonencode and the provider produce, and in an indented form.
func testAccJWKS(t *testing.T, keys ...map[string]any) (compact, indented string) {
	t.Helper()
	set := map[string]any{"keys": keys}
	c, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	i, err := json.MarshalIndent(set, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	return string(c), string(i)
}

func testAccProviderImportStep() resource.TestStep {
	return resource.TestStep{
		ResourceName:      "keycard_provider.test",
		ImportState:       true,
		ImportStateVerify: true,
		ImportStateIdFunc: func(state *terraform.State) (string, error) {
			rs := state.RootModule().Resources["keycard_provider.test"]
			return fmt.Sprintf("zones/%s/providers/%s", rs.Primary.Attributes["zone_id"], rs.Primary.ID), nil
		},
		ImportStateVerifyIgnore: []string{"client_secret"},
	}
}

func TestAccProviderResource_jwksURI(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderResourceConfig_jwksURI(rName, issuer, issuer+"/keys"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.jwks_uri", issuer+"/keys"),
					resource.TestCheckNoResourceAttr("keycard_provider.test", "oauth2.jwks"),
				),
			},
			testAccProviderImportStep(),
			{
				Config: testAccProviderResourceConfig_jwksURI(rName, issuer, issuer+"/keys/v2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("keycard_provider.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.jwks_uri", issuer+"/keys/v2"),
			},
			// Removing jwks_uri from config clears it on the provider.
			{
				Config: testAccProviderResourceConfig_basic(rName, issuer),
				Check:  resource.TestCheckNoResourceAttr("keycard_provider.test", "oauth2.jwks_uri"),
			},
		},
	})
}

func TestAccProviderResource_jwks(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)

	rsaKey := testAccRSAJWK(t, "rsa-1")
	ecKey := testAccECJWK(t, "ec-1")
	oneKey, oneKeyIndented := testAccJWKS(t, rsaKey)
	twoKeys, _ := testAccJWKS(t, rsaKey, ecKey)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderResourceConfig_jwksEncoded(rName, issuer, oneKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.jwks", oneKey),
					resource.TestCheckNoResourceAttr("keycard_provider.test", "oauth2.jwks_uri"),
				),
			},
			testAccProviderImportStep(),
			// The same key set in another format plans no change.
			{
				Config: testAccProviderResourceConfig_jwksHeredoc(rName, issuer, oneKeyIndented),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// Rotating keys updates the provider in place.
			{
				Config: testAccProviderResourceConfig_jwksEncoded(rName, issuer, twoKeys),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("keycard_provider.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.jwks", twoKeys),
			},
			testAccProviderImportStep(),
			// Removing jwks from config clears it on the provider.
			{
				Config: testAccProviderResourceConfig_basic(rName, issuer),
				Check:  resource.TestCheckNoResourceAttr("keycard_provider.test", "oauth2.jwks"),
			},
		},
	})
}

func TestAccProviderResource_switchJwksURIAndJwks(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)
	jwks, _ := testAccJWKS(t, testAccRSAJWK(t, "rsa-1"))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderResourceConfig_jwksURI(rName, issuer, issuer+"/keys"),
				Check:  resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.jwks_uri", issuer+"/keys"),
			},
			{
				Config: testAccProviderResourceConfig_jwksEncoded(rName, issuer, jwks),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("keycard_provider.test", "oauth2.jwks_uri"),
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.jwks", jwks),
				),
			},
			{
				Config: testAccProviderResourceConfig_jwksURI(rName, issuer, issuer+"/keys"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keycard_provider.test", "oauth2.jwks_uri", issuer+"/keys"),
					resource.TestCheckNoResourceAttr("keycard_provider.test", "oauth2.jwks"),
				),
			},
		},
	})
}

func TestAccProviderResource_jwksAndJwksURIConflict(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)
	jwks, _ := testAccJWKS(t, testAccRSAJWK(t, "rsa-1"))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer   = %[2]q
    jwks_uri = "%[2]s/keys"
    jwks     = %[3]q
  }
}
`, rName, issuer, jwks),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func TestAccProviderResource_jwksNotASet(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)
	key, err := json.Marshal([]any{testAccRSAJWK(t, "rsa-1")})
	if err != nil {
		t.Fatal(err)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_jwksEncoded(rName, issuer, string(key)),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Invalid JWKS`),
			},
		},
	})
}

func TestAccProviderResource_jwksPrivateKeyRejected(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	issuer := fmt.Sprintf("https://%s.example.com", rName)
	key := testAccRSAJWK(t, "leaked")
	key["d"] = "AQAB"
	jwks, _ := testAccJWKS(t, key)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderResourceConfig_jwksEncoded(rName, issuer, jwks),
				ExpectError: regexp.MustCompile(`private key material`),
			},
		},
	})
}

func TestAccProviderDataSource_jwksFields(t *testing.T) {
	rName := acctest.RandomWithPrefix("tftest")
	jwks, _ := testAccJWKS(t, testAccECJWK(t, "ec-1"))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "static" {
  name    = "%[1]s-static"
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer = "https://static.%[2]s"
    jwks   = jsonencode(%[3]s)
  }
}

resource "keycard_provider" "uri" {
  name    = "%[1]s-uri"
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer   = "https://uri.%[2]s"
    jwks_uri = "https://uri.%[2]s/keys"
  }
}

data "keycard_provider" "static" {
  zone_id = keycard_provider.static.zone_id
  id      = keycard_provider.static.id
}

data "keycard_provider" "uri" {
  zone_id = keycard_provider.uri.zone_id
  id      = keycard_provider.uri.id
}
`, rName, rName+".example.com", jwks),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_provider.static", "oauth2.jwks", jwks),
					resource.TestCheckNoResourceAttr("data.keycard_provider.static", "oauth2.jwks_uri"),
					resource.TestCheckResourceAttr("data.keycard_provider.uri", "oauth2.jwks_uri", "https://uri."+rName+".example.com/keys"),
					resource.TestCheckNoResourceAttr("data.keycard_provider.uri", "oauth2.jwks"),
				),
			},
		},
	})
}

func testAccProviderResourceConfig_jwksURI(name, issuer, jwksURI string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer   = %[2]q
    jwks_uri = %[3]q
  }
}
`, name, issuer, jwksURI)
}

// testAccProviderResourceConfig_jwksEncoded passes the JWK Set through
// jsonencode, as most configurations will. JSON object syntax is valid HCL.
func testAccProviderResourceConfig_jwksEncoded(name, issuer, jwks string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer = %[2]q
    jwks   = jsonencode(%[3]s)
  }
}
`, name, issuer, jwks)
}

// testAccProviderResourceConfig_jwksHeredoc passes the JWK Set as a literal
// string, as file("jwks.json") would.
func testAccProviderResourceConfig_jwksHeredoc(name, issuer, jwks string) string {
	return testAccOrgZone + fmt.Sprintf(`
resource "keycard_provider" "test" {
  name    = %[1]q
  zone_id = data.keycard_organization.test.zone_id

  oauth2 = {
    issuer = %[2]q
    jwks   = <<-EOT
%[3]s
EOT
  }
}
`, name, issuer, jwks)
}
