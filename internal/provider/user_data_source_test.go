package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccUserID returns a user in the organization's builtin zone. The provider
// cannot create users, so the ID comes from the environment.
func testAccUserID(t *testing.T) string {
	userID := os.Getenv("KEYCARD_TEST_USER_ID")
	if userID == "" {
		t.Skip("KEYCARD_TEST_USER_ID must be set to run user data source tests")
	}
	return userID
}

// testAccExternalUser returns a SCIM-provisioned user that has logged in at
// least once, so subject and issuer are populated.
func testAccExternalUser(t *testing.T) (userID, issuer string) {
	userID = os.Getenv("KEYCARD_TEST_EXTERNAL_USER_ID")
	issuer = os.Getenv("KEYCARD_TEST_EXTERNAL_ISSUER")
	if userID == "" || issuer == "" {
		t.Skip("KEYCARD_TEST_EXTERNAL_USER_ID and KEYCARD_TEST_EXTERNAL_ISSUER must be set to run external user tests")
	}
	return userID, issuer
}

func TestAccUserDataSource_byID(t *testing.T) {
	userID := testAccUserID(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_byID(userID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_user.test", "id", userID),
					resource.TestCheckResourceAttrPair("data.keycard_user.test", "zone_id", testAccOrgZoneRef, "zone_id"),
					resource.TestCheckResourceAttrSet("data.keycard_user.test", "identifier"),
					resource.TestCheckResourceAttrSet("data.keycard_user.test", "email"),
					resource.TestCheckResourceAttrSet("data.keycard_user.test", "status"),
					resource.TestCheckResourceAttrSet("data.keycard_user.test", "external"),
				),
			},
		},
	})
}

// Resolves the user by ID, then again by its own email and issuer, and checks
// both paths land on the same user.
func TestAccUserDataSource_byEmailAndIssuer(t *testing.T) {
	userID := testAccUserID(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_byEmailAndIssuer(userID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_user.by_email", "id", userID),
					resource.TestCheckResourceAttrPair("data.keycard_user.by_email", "email", "data.keycard_user.by_id", "email"),
					resource.TestCheckResourceAttrPair("data.keycard_user.by_email", "issuer", "data.keycard_user.by_id", "issuer"),
				),
			},
		},
	})
}

func TestAccUserDataSource_byIdentifier(t *testing.T) {
	userID := testAccUserID(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_byIdentifier(userID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_user.by_identifier", "id", userID),
				),
			},
		},
	})
}

func TestAccUserDataSource_bySubjectAndIssuer(t *testing.T) {
	userID, issuer := testAccExternalUser(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_bySubjectAndIssuer(userID, issuer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.keycard_user.by_subject", "id", userID),
					resource.TestCheckResourceAttr("data.keycard_user.by_subject", "issuer", issuer),
					resource.TestCheckResourceAttr("data.keycard_user.by_subject", "external", "true"),
				),
			},
		},
	})
}

func TestAccUserDataSource_emailWithoutIssuer(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgZone + `
data "keycard_user" "test" {
  zone_id = data.keycard_organization.test.zone_id
  email   = "nobody@example.com"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func TestAccUserDataSource_idWithIssuer(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgZone + `
data "keycard_user" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = "u1"
  issuer  = "https://idp.example.com"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func TestAccUserDataSource_notFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckBasic(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgZone + `
data "keycard_user" "test" {
  zone_id = data.keycard_organization.test.zone_id
  email   = "nobody-tftest@example.com"
  issuer  = "https://nobody.example.com"
}
`,
				ExpectError: regexp.MustCompile(`User Not Found`),
			},
		},
	})
}

func testAccUserDataSourceConfig_byID(userID string) string {
	return testAccOrgZone + fmt.Sprintf(`
data "keycard_user" "test" {
  zone_id = data.keycard_organization.test.zone_id
  id      = %[1]q
}
`, userID)
}

func testAccUserDataSourceConfig_byEmailAndIssuer(userID string) string {
	return testAccOrgZone + fmt.Sprintf(`
data "keycard_user" "by_id" {
  zone_id = data.keycard_organization.test.zone_id
  id      = %[1]q
}

data "keycard_user" "by_email" {
  zone_id = data.keycard_organization.test.zone_id
  email   = data.keycard_user.by_id.email
  issuer  = data.keycard_user.by_id.issuer
}
`, userID)
}

func testAccUserDataSourceConfig_byIdentifier(userID string) string {
	return testAccOrgZone + fmt.Sprintf(`
data "keycard_user" "by_id" {
  zone_id = data.keycard_organization.test.zone_id
  id      = %[1]q
}

data "keycard_user" "by_identifier" {
  zone_id    = data.keycard_organization.test.zone_id
  identifier = data.keycard_user.by_id.identifier
}
`, userID)
}

func testAccUserDataSourceConfig_bySubjectAndIssuer(userID, issuer string) string {
	return testAccOrgZone + fmt.Sprintf(`
data "keycard_user" "by_id" {
  zone_id = data.keycard_organization.test.zone_id
  id      = %[1]q
}

data "keycard_user" "by_subject" {
  zone_id = data.keycard_organization.test.zone_id
  subject = data.keycard_user.by_id.subject
  issuer  = %[2]q
}
`, userID, issuer)
}
