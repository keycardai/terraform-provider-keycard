package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestApplicationCredentialUpdate(t *testing.T) {
	for _, kind := range []string{"url", "public"} {
		for _, status := range []int{200, 204, 400} {
			t.Run(fmt.Sprintf("%s/%d", kind, status), func(t *testing.T) {
				value := "https://example.com/updated"
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodPatch || req.URL.Path != "/zones/z1/application-credentials/c1" {
						t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					}
					var body map[string]any
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if len(body) != 2 || body["type"] != kind || body["identifier"] != value {
						t.Errorf("unexpected body: %v", body)
					}
					if status == 200 {
						w.Header().Set("Content-Type", "application/json")
					}
					w.WriteHeader(status)
					if status == 200 {
						if err := json.NewEncoder(w).Encode(map[string]string{"id": "c1", "zone_id": "z1", "application_id": "a1", "type": kind, "identifier": value}); err != nil {
							t.Error(err)
						}
					}
				}))
				defer srv.Close()
				var r resource.Resource = &ApplicationURLCredentialResource{client: testClientFor(t, srv)}
				attr := "url"
				if kind == "public" {
					r = &ApplicationPublicCredentialResource{client: testClientFor(t, srv)}
					attr = "identifier"
				}
				var schemaResp resource.SchemaResponse
				r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
				vals := map[string]tftypes.Value{}
				for key, v := range map[string]string{"id": "c1", "zone_id": "z1", "application_id": "a1", attr: value} {
					vals[key] = tftypes.NewValue(tftypes.String, v)
				}
				state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), vals)}
				resp := resource.UpdateResponse{State: state}
				r.Update(context.Background(), resource.UpdateRequest{Plan: tfsdk.Plan(state), State: state}, &resp)
				if resp.Diagnostics.HasError() != (status != 200) {
					t.Fatalf("diagnostics = %v", resp.Diagnostics)
				}
				if status == 200 {
					for key, want := range map[string]string{"id": "c1", attr: value} {
						var got string
						if diags := resp.State.GetAttribute(context.Background(), path.Root(key), &got); diags.HasError() {
							t.Fatal(diags)
						}
						if got != want {
							t.Errorf("%s = %q, want %q", key, got, want)
						}
					}
				}
			})
		}
	}
}
