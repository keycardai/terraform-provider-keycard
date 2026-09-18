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

// groupReadState builds a prior state for the group resource carrying just the
// keys Read needs to issue its GET, mirroring the state ImportState leaves behind.
func groupReadState(t *testing.T, r *GroupResource, zoneID, groupID string) tfsdk.State {
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
	vals["zone_id"] = tftypes.NewValue(tftypes.String, zoneID)
	vals["id"] = tftypes.NewValue(tftypes.String, groupID)

	return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
}

func groupReadServer(t *testing.T, group map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(group); err != nil {
			t.Errorf("encoding response: %s", err)
		}
	}))
}

func TestGroupResourceRead_mapsExternalFields(t *testing.T) {
	srv := groupReadServer(t, map[string]any{
		"id": "g1", "zone_id": "z1", "organization_id": "o1", "identifier": "eng", "name": "Engineering",
		"external": false, "external_issuer": nil,
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
	})
	defer srv.Close()

	r := &GroupResource{client: testClientFor(t, srv)}
	state := groupReadState(t, r, "z1", "g1")
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}

	var data GroupModel
	if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
		t.Fatalf("state: %s", diags)
	}
	if data.External.ValueBool() {
		t.Errorf("external = true, want false")
	}
	if !data.ExternalIssuer.IsNull() {
		t.Errorf("external_issuer = %v, want null", data.ExternalIssuer)
	}
}

func TestGroupResourceRead_rejectsExternalGroup(t *testing.T) {
	srv := groupReadServer(t, map[string]any{
		"id": "g1", "zone_id": "z1", "organization_id": "o1", "identifier": "eng", "name": "Engineering",
		"external": true, "external_issuer": "https://idp.example.com",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
	})
	defer srv.Close()

	r := &GroupResource{client: testClientFor(t, srv)}
	state := groupReadState(t, r, "z1", "g1")
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic for an external group")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "keycard_group") || !strings.Contains(detail, "data source") {
		t.Errorf("diagnostic should point to the keycard_group data source, got: %s", detail)
	}
}
