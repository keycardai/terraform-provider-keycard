package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

const testRSAJWK = `{"kty":"RSA","kid":"k1","use":"sig","alg":"RS256","n":"sXchDaQebHnPiGvyDOAT4saGEUetSyo9MKLOoWFsueri23bOdgWp4Dy1WlUzewbgBHod5pcM9H95GQRV3JDXboIRROSBigeC5yjU1hGzHHyXss8UDprecbAYxknTcQkhslANGRUZmdTOQ5qTRsLAt6BTYuyvVRdhS8exSZEy_c4gs_7svlJJQ4H9_NxsiIoLwAEk7-Q3UXERGYw_75IDrGA84-lA_-Ct4eTlXHBIY2EaV7t7LjJaynVJCpkv4LKjTTAumiGUIuQhrNhZLuF_RJLqHpM2kgWFLU7-VTdL1VbC2tejvcI2BlMkEpk1BzBZI0KQB0GaDWFLN-aEAw3vRw","e":"AQAB"}`

func TestJwksSetValidator(t *testing.T) {
	cases := map[string]struct {
		value   types.String
		wantErr string
	}{
		"null":           {value: types.StringNull()},
		"unknown":        {value: types.StringUnknown()},
		"valid":          {value: types.StringValue(`{"keys":[` + testRSAJWK + `]}`)},
		"invalid json":   {value: types.StringValue(`{"keys":`)}, // reported by jsontypes
		"bare array":     {value: types.StringValue(`[` + testRSAJWK + `]`), wantErr: "bare array"},
		"null literal":   {value: types.StringValue(`null`), wantErr: "bare array"},
		"no keys member": {value: types.StringValue(`{}`), wantErr: `must contain a "keys" array`},
		"extra member":   {value: types.StringValue(`{"keys":[` + testRSAJWK + `],"foo":1}`), wantErr: `"foo"`},
		"keys not array": {value: types.StringValue(`{"keys":{}}`), wantErr: "array of JSON objects"},
		"empty keys":     {value: types.StringValue(`{"keys":[]}`), wantErr: "at least one key"},
		"null keys":      {value: types.StringValue(`{"keys":null}`), wantErr: "at least one key"},
		"key not object": {value: types.StringValue(`{"keys":["x"]}`), wantErr: "array of JSON objects"},
		"null key":       {value: types.StringValue(`{"keys":[null]}`), wantErr: "keys[0]"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("oauth2").AtName("jwks"), ConfigValue: tc.value}
			var resp validator.StringResponse
			jwksSetValidator{}.ValidateString(context.Background(), req, &resp)

			if tc.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected error: %s", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected an error containing %q", tc.wantErr)
			}
			if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, tc.wantErr) {
				t.Errorf("error %q does not contain %q", detail, tc.wantErr)
			}
		})
	}
}

func TestJwksRoundTrip_keepsKeyMembersInCanonicalForm(t *testing.T) {
	in := jsontypes.NewNormalizedValue("{\n  \"keys\": [\n    " + testRSAJWK + ",\n    {\"y\": \"yy\", \"x\": \"xx\", \"crv\": \"P-256\", \"kid\": \"k2\", \"kty\": \"EC\"}\n  ]\n}")

	jwks, diags := jwksFromTerraform(in)
	if diags.HasError() {
		t.Fatalf("decode: %s", diags)
	}
	if len(jwks.Keys) != 2 || jwks.Keys[1].Kid != "k2" {
		t.Fatalf("decoded keys = %+v", jwks.Keys)
	}
	if v, ok := jwks.Keys[0].Get("n"); !ok || v == "" {
		t.Errorf("RSA modulus not kept as an additional property")
	}

	out, diags := jwksToTerraform(nullable.NewNullableWithValue(jwks))
	if diags.HasError() {
		t.Fatalf("encode: %s", diags)
	}

	// Compact, with members sorted, as Terraform's jsonencode produces.
	want := `{"keys":[{"alg":"RS256","e":"AQAB","kid":"k1","kty":"RSA","n":"` + rsaModulus(t) + `","use":"sig"},{"crv":"P-256","kid":"k2","kty":"EC","x":"xx","y":"yy"}]}`
	if out.ValueString() != want {
		t.Errorf("encoded JWKS\n got: %s\nwant: %s", out.ValueString(), want)
	}

	equal, diags := out.StringSemanticEquals(context.Background(), in)
	if diags.HasError() || !equal {
		t.Errorf("round trip is not semantically equal to the input")
	}
}

func rsaModulus(t *testing.T) string {
	t.Helper()
	var key map[string]string
	if err := json.Unmarshal([]byte(testRSAJWK), &key); err != nil {
		t.Fatal(err)
	}
	return key["n"]
}

func TestJwksToTerraform_unsetIsNull(t *testing.T) {
	for name, val := range map[string]nullable.Nullable[client.ProviderJwks]{
		"absent":        {},
		"explicit null": nullable.NewNullNullable[client.ProviderJwks](),
	} {
		t.Run(name, func(t *testing.T) {
			out, diags := jwksToTerraform(val)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}
			if !out.IsNull() {
				t.Errorf("got %s, want null", out)
			}
		})
	}
}

func TestClearableUpdateValues(t *testing.T) {
	set := jsontypes.NewNormalizedValue(`{"keys":[` + testRSAJWK + `]}`)
	unset := jsontypes.NewNormalizedNull()

	cases := map[string]struct {
		planned, prior    jsontypes.Normalized
		specified, isNull bool
	}{
		"never set":           {planned: unset, prior: unset},
		"unknown":             {planned: jsontypes.NewNormalizedUnknown(), prior: set},
		"removed from config": {planned: unset, prior: set, specified: true, isNull: true},
		"set":                 {planned: set, prior: unset, specified: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			jwks, diags := jwksUpdateValue(tc.planned, tc.prior)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}
			if jwks.IsSpecified() != tc.specified || jwks.IsNull() != tc.isNull {
				t.Errorf("jwks: specified=%v null=%v, want specified=%v null=%v", jwks.IsSpecified(), jwks.IsNull(), tc.specified, tc.isNull)
			}

			uri := clearableStringUpdateValue(tc.planned.StringValue, tc.prior.StringValue)
			if uri.IsSpecified() != tc.specified || uri.IsNull() != tc.isNull {
				t.Errorf("jwks_uri: specified=%v null=%v, want specified=%v null=%v", uri.IsSpecified(), uri.IsNull(), tc.specified, tc.isNull)
			}
		})
	}
}

func TestJwksSemanticEqualityModifier(t *testing.T) {
	state := types.StringValue(`{"keys":[{"kid":"k1","kty":"RSA"}]}`)
	cases := map[string]struct {
		config types.String
		want   types.String
	}{
		"reformatted keeps state": {config: types.StringValue("{\"keys\": [ {\"kty\": \"RSA\", \"kid\": \"k1\"} ]}"), want: state},
		"changed keeps config":    {config: types.StringValue(`{"keys":[{"kid":"k2","kty":"RSA"}]}`), want: types.StringValue(`{"keys":[{"kid":"k2","kty":"RSA"}]}`)},
		"removed keeps null":      {config: types.StringNull(), want: types.StringNull()},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := planmodifier.StringRequest{ConfigValue: tc.config, PlanValue: tc.config, StateValue: state}
			resp := planmodifier.StringResponse{PlanValue: tc.config}
			jwksSemanticEqualityModifier{}.PlanModifyString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("plan = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}

// providerResponse returns a provider API response with the given oauth2 protocol.
func providerResponse(oauth2 map[string]any) map[string]any {
	return map[string]any{
		"id": "p1", "zone_id": "z1", "organization_id": "o1", "slug": "p1",
		"identifier": "https://issuer.example.com", "name": "Test",
		"protocols":  map[string]any{"oauth2": oauth2},
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
	}
}

func TestProviderOAuth2FromAPIResponse_mapsJwksFields(t *testing.T) {
	var provider client.Provider
	body, _ := json.Marshal(providerResponse(map[string]any{
		"issuer":   "https://issuer.example.com",
		"jwks_uri": nil,
		"jwks":     map[string]any{"keys": []any{map[string]any{"kty": "EC", "kid": "k2", "crv": "P-256", "x": "xx", "y": "yy"}}},
	}))
	if err := json.Unmarshal(body, &provider); err != nil {
		t.Fatal(err)
	}

	obj, diags := providerOAuth2FromAPIResponse(context.Background(), &provider)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	var model OAuth2ProviderModel
	if diags := obj.As(context.Background(), &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("object: %s", diags)
	}
	if !model.JwksURI.IsNull() {
		t.Errorf("jwks_uri = %s, want null", model.JwksURI)
	}
	if want := `{"keys":[{"crv":"P-256","kid":"k2","kty":"EC","x":"xx","y":"yy"}]}`; model.Jwks.ValueString() != want {
		t.Errorf("jwks = %s, want %s", model.Jwks.ValueString(), want)
	}
}

// TestProviderResourceUpdate_switchesFromJwksURIToJwks checks the PATCH body
// when a provider moves from jwks_uri to a static JWKS: jwks_uri is cleared
// and jwks is set in the same request.
func TestProviderResourceUpdate_switchesFromJwksURIToJwks(t *testing.T) {
	ctx := context.Background()
	jwks := `{"keys":[` + testRSAJWK + `]}`

	var patch struct {
		Protocols struct {
			Oauth2 map[string]json.RawMessage `json:"oauth2"`
		} `json:"protocols"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPatch {
			t.Errorf("unexpected %s request", req.Method)
		}
		raw, _ := io.ReadAll(req.Body)
		if err := json.Unmarshal(raw, &patch); err != nil {
			t.Errorf("decoding request: %s", err)
		}
		var jwksBody any
		_ = json.Unmarshal([]byte(jwks), &jwksBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(providerResponse(map[string]any{
			"issuer": "https://issuer.example.com", "jwks": jwksBody,
		}))
	}))
	defer srv.Close()

	r := &ProviderResource{client: testClientFor(t, srv)}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	model := func(uri types.String, keys jsontypes.Normalized) ProviderResourceModel {
		oauth2 := OAuth2ProviderModel{
			Issuer:                types.StringValue("https://issuer.example.com"),
			AuthorizationEndpoint: types.StringNull(),
			TokenEndpoint:         types.StringNull(),
			JwksURI:               uri,
			Jwks:                  keys,
		}
		obj, diags := types.ObjectValueFrom(ctx, oauth2.AttributeTypes(), oauth2)
		if diags.HasError() {
			t.Fatalf("oauth2: %s", diags)
		}
		return ProviderResourceModel{
			ID: types.StringValue("p1"), ZoneID: types.StringValue("z1"), Name: types.StringValue("Test"),
			Description: types.StringNull(), Identifier: types.StringValue("https://issuer.example.com"),
			ClientID: types.StringNull(), ClientSecret: types.StringNull(), OAuth2: obj,
		}
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, model(types.StringValue("https://issuer.example.com/jwks"), jsontypes.NewNormalizedNull())); diags.HasError() {
		t.Fatalf("state: %s", diags)
	}
	if diags := plan.Set(ctx, model(types.StringNull(), jsontypes.NewNormalizedValue(jwks))); diags.HasError() {
		t.Fatalf("plan: %s", diags)
	}

	resp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	oauth2 := patch.Protocols.Oauth2
	if v, ok := oauth2["jwks_uri"]; !ok || string(v) != "null" {
		t.Errorf("jwks_uri in request = %s (present %v), want explicit null", v, ok)
	}
	var sent client.ProviderJwks
	if err := json.Unmarshal(oauth2["jwks"], &sent); err != nil {
		t.Fatalf("decoding jwks in request: %s", err)
	}
	if len(sent.Keys) != 1 {
		t.Fatalf("jwks in request = %s, want one key", oauth2["jwks"])
	}
	if n, _ := sent.Keys[0].Get("n"); n != rsaModulus(t) {
		t.Errorf("jwks in request = %s, want the configured key set", oauth2["jwks"])
	}
	if _, ok := oauth2["authorization_endpoint"]; ok {
		t.Errorf("authorization_endpoint should be omitted when never set")
	}
}
