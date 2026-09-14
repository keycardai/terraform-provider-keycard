package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func externalSyncTokenValue(t *testing.T, r *ExternalSyncTokenResource, id, token any) tfsdk.State {
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
		"id":      tftypes.NewValue(tftypes.String, id),
		"zone_id": tftypes.NewValue(tftypes.String, "z1"),
		"token":   tftypes.NewValue(tftypes.String, token),
	}
	return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
}

func externalSyncTokenJSON(withToken bool) map[string]any {
	m := map[string]any{
		"id": "t1", "organization_id": "o1", "zone_id": "z1", "provider_id": "p1",
		"last_used_at": nil,
		"created_at":   "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
	}
	if withToken {
		m["token"] = "secret-token"
	}
	return m
}

// externalSyncTokenServer answers each method with the given status and body
// and records the request paths it saw.
func externalSyncTokenServer(t *testing.T, status int, body map[string]any, paths *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		*paths = append(*paths, req.Method+" "+req.URL.Path)
		if body != nil {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Errorf("encoding response: %s", err)
			}
		}
	}))
}

func externalSyncTokenState(t *testing.T, state tfsdk.State) ExternalSyncTokenModel {
	t.Helper()
	var data ExternalSyncTokenModel
	if diags := state.Get(context.Background(), &data); diags.HasError() {
		t.Fatalf("state: %s", diags)
	}
	return data
}

func TestExternalSyncTokenCreate_storesToken(t *testing.T) {
	var paths []string
	srv := externalSyncTokenServer(t, http.StatusOK, externalSyncTokenJSON(true), &paths)
	defer srv.Close()

	r := &ExternalSyncTokenResource{client: testClientFor(t, srv)}
	plan := externalSyncTokenValue(t, r, tftypes.UnknownValue, tftypes.UnknownValue)
	resp := resource.CreateResponse{State: plan}
	r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	if len(paths) != 1 || paths[0] != "POST /zones/z1/external-sync-tokens" {
		t.Errorf("requests = %v, want single POST", paths)
	}
	data := externalSyncTokenState(t, resp.State)
	if data.ID.ValueString() != "t1" || data.ZoneID.ValueString() != "z1" || data.Token.ValueString() != "secret-token" {
		t.Errorf("state = %+v, want id t1, zone z1, token secret-token", data)
	}
}

func TestExternalSyncTokenCreate_surfacesAPIError(t *testing.T) {
	var paths []string
	srv := externalSyncTokenServer(t, http.StatusBadRequest, map[string]any{"message": "external sync is not enabled"}, &paths)
	defer srv.Close()

	r := &ExternalSyncTokenResource{client: testClientFor(t, srv)}
	plan := externalSyncTokenValue(t, r, tftypes.UnknownValue, tftypes.UnknownValue)
	resp := resource.CreateResponse{State: plan}
	r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan(plan)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error diagnostics")
	}
	if got := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(got, "external sync is not enabled") {
		t.Errorf("error detail = %q, want API message", got)
	}
}

func TestExternalSyncTokenRead_keepsStoredToken(t *testing.T) {
	var paths []string
	srv := externalSyncTokenServer(t, http.StatusOK, externalSyncTokenJSON(false), &paths)
	defer srv.Close()

	r := &ExternalSyncTokenResource{client: testClientFor(t, srv)}
	state := externalSyncTokenValue(t, r, "t1", "secret-token")
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	if len(paths) != 1 || paths[0] != "GET /zones/z1/external-sync-tokens/t1" {
		t.Errorf("requests = %v, want single GET", paths)
	}
	if data := externalSyncTokenState(t, resp.State); data.Token.ValueString() != "secret-token" {
		t.Errorf("token = %q, want stored value preserved", data.Token.ValueString())
	}
}

func TestExternalSyncTokenRead_removesOn404(t *testing.T) {
	var paths []string
	srv := externalSyncTokenServer(t, http.StatusNotFound, map[string]any{"message": "not found"}, &paths)
	defer srv.Close()

	r := &ExternalSyncTokenResource{client: testClientFor(t, srv)}
	state := externalSyncTokenValue(t, r, "t1", "secret-token")
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("state not removed after 404")
	}
}

func TestExternalSyncTokenDelete_treats404AsSuccess(t *testing.T) {
	for status, body := range map[int]map[string]any{
		http.StatusNoContent: nil,
		http.StatusNotFound:  {"message": "not found"},
	} {
		var paths []string
		srv := externalSyncTokenServer(t, status, body, &paths)
		r := &ExternalSyncTokenResource{client: testClientFor(t, srv)}
		state := externalSyncTokenValue(t, r, "t1", "secret-token")
		resp := resource.DeleteResponse{State: state}
		r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
		srv.Close()
		if resp.Diagnostics.HasError() {
			t.Errorf("status %d: unexpected diagnostics: %s", status, resp.Diagnostics)
		}
		if len(paths) != 1 || paths[0] != "DELETE /zones/z1/external-sync-tokens/t1" {
			t.Errorf("status %d: requests = %v, want single DELETE", status, paths)
		}
	}
}
