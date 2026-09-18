package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// ssoTestEndpoint has an "api." host so login_url derivation works; requests
// are rewritten to the httptest server by redirectTransport.
const ssoTestEndpoint = "https://api.test.invalid"

type redirectTransport struct{ target string }

func (t redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u := *req.URL
	u.Scheme = "http"
	u.Host = strings.TrimPrefix(t.target, "http://")
	req.URL = &u
	return http.DefaultTransport.RoundTrip(req)
}

// ssoCall is one recorded request against the fake API.
type ssoCall struct {
	Method string
	Path   string
	Body   map[string]any
}

// ssoServer fakes the org list, SSO connection, and org zone endpoints. It
// records every call and tracks the zone's external_sync_enabled value.
func ssoServer(t *testing.T, syncEnabled bool, calls *[]ssoCall) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		if req.Body != nil {
			raw, _ := io.ReadAll(req.Body)
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Errorf("decoding body %q: %s", raw, err)
				}
			}
		}
		*calls = append(*calls, ssoCall{Method: req.Method, Path: req.URL.Path, Body: body})

		var out any
		switch {
		case req.URL.Path == "/organizations":
			out = map[string]any{"items": []map[string]any{{"id": "org1", "zone_id": "z1"}}}
		case strings.HasSuffix(req.URL.Path, "/sso-connection"):
			if req.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if req.Method == http.MethodPost {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
			}
			out = map[string]any{
				"id": "sso1", "identifier": "https://idp.example.com", "client_id": "cid",
				"client_secret_set": false,
				"created_at":        "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			}
		case req.URL.Path == "/zones/z1":
			if req.Method == http.MethodPatch {
				if v, ok := body["external_sync_enabled"].(bool); ok {
					syncEnabled = v
				}
			}
			out = zoneJSON("p1", syncEnabled)
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out); err != nil {
			t.Errorf("encoding response: %s", err)
		}
	}))
}

func ssoTestResource(t *testing.T, srv *httptest.Server) *SSOConnectionResource {
	t.Helper()
	c, err := client.NewClientWithResponses(ssoTestEndpoint, client.WithHTTPClient(&http.Client{Transport: redirectTransport{target: srv.URL}}))
	if err != nil {
		t.Fatalf("building client: %s", err)
	}
	return &SSOConnectionResource{client: c}
}

// ssoConnectionValue builds a fully-known resource value with client_secret null.
func ssoConnectionValue(t *testing.T, r *SSOConnectionResource, syncEnabled bool) tfsdk.State {
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
	vals := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(attrType, nil)
	}
	vals["id"] = tftypes.NewValue(tftypes.String, "sso1")
	vals["identifier"] = tftypes.NewValue(tftypes.String, "https://idp.example.com")
	vals["client_id"] = tftypes.NewValue(tftypes.String, "cid")
	vals["login_url"] = tftypes.NewValue(tftypes.String, "https://id.test.invalid/openid/connect/login")
	vals["external_sync_enabled"] = tftypes.NewValue(tftypes.Bool, syncEnabled)
	return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
}

func zonePatches(calls []ssoCall) []ssoCall {
	var out []ssoCall
	for _, c := range calls {
		if c.Method == http.MethodPatch && c.Path == "/zones/z1" {
			out = append(out, c)
		}
	}
	return out
}

func TestSSOConnectionCreate_enablesOrgZoneSync(t *testing.T) {
	var calls []ssoCall
	srv := ssoServer(t, false, &calls)
	defer srv.Close()

	r := ssoTestResource(t, srv)
	plan := ssoConnectionValue(t, r, true)
	resp := resource.CreateResponse{State: plan}
	r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	patches := zonePatches(calls)
	if len(patches) != 1 || patches[0].Body["external_sync_enabled"] != true {
		t.Fatalf("zone PATCHes = %+v, want one enabling external sync", patches)
	}

	var data SSOConnectionResourceModel
	if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
		t.Fatalf("state: %s", diags)
	}
	if !data.ExternalSyncEnabled.ValueBool() {
		t.Errorf("external_sync_enabled = false, want true")
	}
}

func TestSSOConnectionCreate_defaultFalseSkipsZonePatch(t *testing.T) {
	var calls []ssoCall
	srv := ssoServer(t, false, &calls)
	defer srv.Close()

	r := ssoTestResource(t, srv)
	plan := ssoConnectionValue(t, r, false)
	resp := resource.CreateResponse{State: plan}
	r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}
	if patches := zonePatches(calls); len(patches) != 0 {
		t.Errorf("zone PATCHes = %+v, want none", patches)
	}
}

func TestSSOConnectionRead_mapsOrgZoneSync(t *testing.T) {
	var calls []ssoCall
	srv := ssoServer(t, true, &calls)
	defer srv.Close()

	r := ssoTestResource(t, srv)
	state := ssoConnectionValue(t, r, false)
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	var data SSOConnectionResourceModel
	if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
		t.Fatalf("state: %s", diags)
	}
	if !data.ExternalSyncEnabled.ValueBool() {
		t.Errorf("external_sync_enabled = false, want true from zone")
	}
}

func TestSSOConnectionDelete_disablesSyncBeforeDisablingSSO(t *testing.T) {
	var calls []ssoCall
	srv := ssoServer(t, true, &calls)
	defer srv.Close()

	r := ssoTestResource(t, srv)
	state := ssoConnectionValue(t, r, true)
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	patchIdx, deleteIdx := -1, -1
	for i, c := range calls {
		switch {
		case c.Method == http.MethodPatch && c.Path == "/zones/z1":
			patchIdx = i
			if c.Body["external_sync_enabled"] != false {
				t.Errorf("zone PATCH body = %v, want external_sync_enabled false", c.Body)
			}
		case c.Method == http.MethodDelete:
			deleteIdx = i
		}
	}
	if patchIdx == -1 || deleteIdx == -1 || patchIdx > deleteIdx {
		t.Errorf("calls = %+v, want zone PATCH before SSO DELETE", calls)
	}
}
