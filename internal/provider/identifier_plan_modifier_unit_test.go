package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// TestSyncOAuth2WithIdentifierModifier_clearsKeySources checks the plan when
// the oauth2 block is removed from config: the block carries forward from
// state, but jwks_uri and jwks are cleared, so the provider stops trusting
// keys the configuration no longer names.
func TestSyncOAuth2WithIdentifierModifier_clearsKeySources(t *testing.T) {
	ctx := context.Background()
	const issuer = "https://issuer.example.com"
	jwks := jsontypes.NewNormalizedValue(`{"keys":[` + testRSAJWK + `]}`)

	var schemaResp resource.SchemaResponse
	(&ProviderResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	model := func(identifier types.String, oauth2 types.Object) ProviderResourceModel {
		return ProviderResourceModel{
			ID: types.StringValue("p1"), ZoneID: types.StringValue("z1"), Name: types.StringValue("Test"),
			Description: types.StringNull(), Identifier: identifier,
			ClientID: types.StringNull(), ClientSecret: types.StringNull(), OAuth2: oauth2,
		}
	}
	oauth2Obj := func(m OAuth2ProviderModel) types.Object {
		obj, diags := types.ObjectValueFrom(ctx, m.AttributeTypes(), m)
		if diags.HasError() {
			t.Fatalf("oauth2: %s", diags)
		}
		return obj
	}

	cases := map[string]struct {
		stateURI   types.String
		stateJwks  jsontypes.Normalized
		identifier types.String
		wantIssuer string
	}{
		"jwks, same identifier": {
			stateURI: types.StringNull(), stateJwks: jwks,
			identifier: types.StringValue(issuer), wantIssuer: issuer,
		},
		"jwks_uri, new identifier": {
			stateURI: types.StringValue(issuer + "/keys"), stateJwks: jsontypes.NewNormalizedNull(),
			identifier: types.StringValue("https://new.example.com"), wantIssuer: "https://new.example.com",
		},
		"jwks, unknown identifier": {
			stateURI: types.StringNull(), stateJwks: jwks,
			identifier: types.StringUnknown(), wantIssuer: issuer,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			stateOAuth2 := oauth2Obj(OAuth2ProviderModel{
				Issuer:                types.StringValue(issuer),
				AuthorizationEndpoint: types.StringValue(issuer + "/authorize"),
				TokenEndpoint:         types.StringNull(),
				JwksURI:               tc.stateURI,
				Jwks:                  tc.stateJwks,
			})

			state := tfsdk.State{Schema: schemaResp.Schema}
			if diags := state.Set(ctx, model(types.StringValue(issuer), stateOAuth2)); diags.HasError() {
				t.Fatalf("state: %s", diags)
			}
			// tfsdk.Config has no setter, so build the config through state.
			configState := tfsdk.State{Schema: schemaResp.Schema}
			nullOAuth2 := types.ObjectNull(OAuth2ProviderModel{}.AttributeTypes())
			if diags := configState.Set(ctx, model(tc.identifier, nullOAuth2)); diags.HasError() {
				t.Fatalf("config: %s", diags)
			}

			req := planmodifier.ObjectRequest{
				Path:        path.Root("oauth2"),
				Config:      tfsdk.Config(configState),
				ConfigValue: nullOAuth2,
				State:       state,
				StateValue:  stateOAuth2,
				PlanValue:   types.ObjectUnknown(OAuth2ProviderModel{}.AttributeTypes()),
			}
			resp := planmodifier.ObjectResponse{PlanValue: req.PlanValue}
			syncOAuth2WithIdentifierModifier{}.PlanModifyObject(ctx, req, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
			}

			var planned OAuth2ProviderModel
			if diags := resp.PlanValue.As(ctx, &planned, basetypes.ObjectAsOptions{}); diags.HasError() {
				t.Fatalf("plan: %s", diags)
			}
			if !planned.Jwks.IsNull() || !planned.JwksURI.IsNull() {
				t.Errorf("jwks = %s, jwks_uri = %s, want both null", planned.Jwks, planned.JwksURI)
			}
			if planned.Issuer.ValueString() != tc.wantIssuer {
				t.Errorf("issuer = %s, want %s", planned.Issuer, tc.wantIssuer)
			}
			if planned.AuthorizationEndpoint.ValueString() != issuer+"/authorize" {
				t.Errorf("authorization_endpoint = %s, want it carried forward", planned.AuthorizationEndpoint)
			}
		})
	}
}
