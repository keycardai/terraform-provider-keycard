package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// zoneUserIdentityConfigValue builds a fully-known value for the resource
// schema, usable as plan or prior state.
func zoneUserIdentityConfigValue(t *testing.T, r *ZoneUserIdentityConfigResource, zoneID, providerID string, syncEnabled bool) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %s", schemaResp.Diagnostics)
	}

	objType, ok := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not an object")
	}
	vals := map[string]tftypes.Value{
		"zone_id":               tftypes.NewValue(tftypes.String, zoneID),
		"provider_id":           tftypes.NewValue(tftypes.String, providerID),
		"external_sync_enabled": tftypes.NewValue(tftypes.Bool, syncEnabled),
	}
	return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
}

// zoneServer records every PATCH body and answers each request with zone.
func zoneServer(t *testing.T, zone map[string]any, patches *[]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPatch {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("reading body: %s", err)
			}
			var patch map[string]any
			if err := json.Unmarshal(body, &patch); err != nil {
				t.Errorf("decoding body %q: %s", body, err)
			}
			*patches = append(*patches, patch)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(zone); err != nil {
			t.Errorf("encoding response: %s", err)
		}
	}))
}

func zoneJSON(providerID any, syncEnabled bool) map[string]any {
	return map[string]any{
		"id": "z1", "organization_id": "o1", "slug": "z1", "name": "Zone",
		"user_identity_provider_id": providerID,
		"external_sync_enabled":     syncEnabled,
		"protocols": map[string]any{
			"oauth2": map[string]any{
				"issuer": "https://z1.example.com", "redirect_uri": "https://z1.example.com/cb",
				"pkce_required": true, "dcr_enabled": true,
			},
			"openid": map[string]any{
				"provider_configuration": "https://z1.example.com/.well-known/openid-configuration",
				"userinfo_endpoint":      "https://z1.example.com/userinfo",
			},
		},
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
	}
}

func TestZoneUserIdentityConfigCreate_sendsProviderAndSyncTogether(t *testing.T) {
	var patches []map[string]any
	srv := zoneServer(t, zoneJSON("p1", true), &patches)
	defer srv.Close()

	r := &ZoneUserIdentityConfigResource{client: testClientFor(t, srv)}
	plan := zoneUserIdentityConfigValue(t, r, "z1", "p1", true)
	resp := resource.CreateResponse{State: plan}
	r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	if len(patches) != 1 {
		t.Fatalf("got %d PATCH requests, want 1", len(patches))
	}
	if patches[0]["user_identity_provider_id"] != "p1" || patches[0]["external_sync_enabled"] != true {
		t.Errorf("PATCH body = %v, want provider p1 and external_sync_enabled true", patches[0])
	}

	var data ZoneUserIdentityConfigResourceModel
	if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
		t.Fatalf("state: %s", diags)
	}
	if !data.ExternalSyncEnabled.ValueBool() {
		t.Errorf("external_sync_enabled = false, want true")
	}
}

func TestZoneUserIdentityConfigRead_mapsExternalSyncEnabled(t *testing.T) {
	var patches []map[string]any
	srv := zoneServer(t, zoneJSON("p1", true), &patches)
	defer srv.Close()

	r := &ZoneUserIdentityConfigResource{client: testClientFor(t, srv)}
	state := zoneUserIdentityConfigValue(t, r, "z1", "p1", false)
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	var data ZoneUserIdentityConfigResourceModel
	if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
		t.Fatalf("state: %s", diags)
	}
	if !data.ExternalSyncEnabled.ValueBool() {
		t.Errorf("external_sync_enabled = false, want true from API")
	}
}

func TestZoneUserIdentityConfigDelete_disablesSyncWhileUnsettingProvider(t *testing.T) {
	var patches []map[string]any
	srv := zoneServer(t, zoneJSON(nil, false), &patches)
	defer srv.Close()

	r := &ZoneUserIdentityConfigResource{client: testClientFor(t, srv)}
	state := zoneUserIdentityConfigValue(t, r, "z1", "p1", true)
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	if len(patches) != 1 {
		t.Fatalf("got %d PATCH requests, want 1", len(patches))
	}
	pid, present := patches[0]["user_identity_provider_id"]
	if !present || pid != nil {
		t.Errorf("user_identity_provider_id = %v (present=%t), want explicit null", pid, present)
	}
	if patches[0]["external_sync_enabled"] != false {
		t.Errorf("external_sync_enabled = %v, want false", patches[0]["external_sync_enabled"])
	}
}
