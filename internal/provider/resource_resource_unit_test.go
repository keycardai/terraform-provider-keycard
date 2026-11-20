package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResourceCredentialLifetimeSecondsValidation(t *testing.T) {
	ctx := context.Background()
	var resp resource.SchemaResponse
	NewResourceResource().Schema(ctx, resource.SchemaRequest{}, &resp)
	attribute, ok := resp.Schema.Attributes["credential_lifetime_seconds"].(schema.Int64Attribute)
	if !ok {
		t.Fatal("credential_lifetime_seconds is not an Int64 attribute")
	}
	for _, value := range []types.Int64{types.Int64Null(), types.Int64Unknown(), types.Int64Value(60), types.Int64Value(86400), types.Int64Value(59), types.Int64Value(86401)} {
		t.Run(value.String(), func(t *testing.T) {
			var result validator.Int64Response
			for _, v := range attribute.Validators {
				v.ValidateInt64(ctx, validator.Int64Request{ConfigValue: value}, &result)
			}
			wantError := !value.IsNull() && !value.IsUnknown() && (value.ValueInt64() < 60 || value.ValueInt64() > 86400)
			if result.Diagnostics.HasError() != wantError {
				t.Fatalf("diagnostics = %s, want error %v", result.Diagnostics, wantError)
			}
		})
	}
}

func TestResourceCredentialLifetimeSecondsRequests(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, tc := range []struct {
			name  string
			value types.Int64
			want  string
		}{
			{name: "set", value: types.Int64Value(3600), want: "3600"},
			{name: "unset", value: types.Int64Null(), want: "null"},
			{name: "unknown", value: types.Int64Unknown()},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				ctx := context.Background()
				var body map[string]json.RawMessage
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					value := body["credential_lifetime_seconds"]
					if len(value) == 0 {
						value = json.RawMessage("null")
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": "r1", "zone_id": "z1", "credential_lifetime_seconds": value})
				}))
				defer srv.Close()
				r := &ResourceResource{client: testClientFor(t, srv)}
				var schemaResp resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
				model := ResourceModel{
					ID: types.StringValue("r1"), ZoneID: types.StringValue("z1"),
					Name: types.StringValue("Test"), Identifier: types.StringValue("https://example.com"),
					CredentialProviderID: types.StringValue("p1"), CredentialLifetimeSeconds: tc.value,
					Metadata: types.ObjectNull(ResourceMetadataModel{}.AttributeTypes()),
					OAuth2:   types.ObjectNull(ResourceOAuth2Model{}.AttributeTypes()),
				}
				plan := tfsdk.Plan{Schema: schemaResp.Schema}
				if diags := plan.Set(ctx, model); diags.HasError() {
					t.Fatal(diags)
				}
				state := tfsdk.State{Schema: schemaResp.Schema}
				if operation == "create" {
					resp := resource.CreateResponse{State: state}
					r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
					if resp.Diagnostics.HasError() {
						t.Fatal(resp.Diagnostics)
					}
					state = resp.State
				} else {
					resp := resource.UpdateResponse{State: state}
					r.Update(ctx, resource.UpdateRequest{Plan: plan}, &resp)
					if resp.Diagnostics.HasError() {
						t.Fatal(resp.Diagnostics)
					}
					state = resp.State
				}
				want := tc.want
				if operation == "create" && tc.value.IsNull() {
					want = ""
				}
				if got := string(body["credential_lifetime_seconds"]); got != want {
					t.Errorf("request lifetime = %q, want %q", got, want)
				}
				var result ResourceModel
				if diags := state.Get(ctx, &result); diags.HasError() {
					t.Fatal(diags)
				}
				expected := tc.value
				if expected.IsUnknown() {
					expected = types.Int64Null()
				}
				if !result.CredentialLifetimeSeconds.Equal(expected) {
					t.Errorf("state lifetime = %s, want %s", result.CredentialLifetimeSeconds, expected)
				}
			})
		}
	}
}
