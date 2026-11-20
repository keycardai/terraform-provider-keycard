package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

func TestApplicationPublicKeyCredentialMappingRejectsWrongType(t *testing.T) {
	for _, credentialType := range []string{"url", "password", ""} {
		t.Run(credentialType, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"id": "other", "type": credentialType})
			if err != nil {
				t.Fatal(err)
			}
			var createResponse client.ApplicationCredentialCreateResponse
			var readResponse client.ApplicationCredential
			if err := json.Unmarshal(body, &createResponse); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(body, &readResponse); err != nil {
				t.Fatal(err)
			}
			for name, mapResponse := range map[string]func(*ApplicationPublicKeyCredentialModel) diag.Diagnostics{
				"create": func(data *ApplicationPublicKeyCredentialModel) diag.Diagnostics {
					return updateApplicationPublicKeyCredentialModelFromCreateResponse(&createResponse, data)
				},
				"read": func(data *ApplicationPublicKeyCredentialModel) diag.Diagnostics {
					return updateApplicationPublicKeyCredentialModelFromAPIResponse(&readResponse, data)
				},
			} {
				t.Run(name, func(t *testing.T) {
					data := ApplicationPublicKeyCredentialModel{ID: types.StringValue("original")}
					diags := mapResponse(&data)
					if len(diags) != 1 || diags[0].Summary() != "API Error" || diags[0].Detail() != fmt.Sprintf("Expected public key credential, got type %q", credentialType) {
						t.Fatalf("unexpected diagnostics: %v", diags)
					}
					if data.ID.ValueString() != "original" {
						t.Fatal("wrong-type response changed the model")
					}
				})
			}
		})
	}
}

func publicKeyCredentialState(t *testing.T, r *ApplicationPublicKeyCredentialResource, identifier any) tfsdk.State {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	vals := map[string]tftypes.Value{}
	for key, value := range map[string]any{"id": "c1", "zone_id": "z1", "application_id": "a1", "jwks_uri": "https://example.com/jwks.json", "identifier": identifier} {
		vals[key] = tftypes.NewValue(tftypes.String, value)
	}
	return tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(context.Background()), vals)}
}

func TestApplicationPublicKeyCredentialCreate(t *testing.T) {
	for _, identifier := range []any{nil, tftypes.UnknownValue, "custom-client"} {
		t.Run(fmt.Sprint(identifier), func(t *testing.T) {
			wantID := "generated-client"
			if identifier == "custom-client" {
				wantID = "custom-client"
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPost || req.URL.Path != "/zones/z1/application-credentials" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["type"] != "public-key" || body["application_id"] != "a1" || body["jwks_uri"] != "https://example.com/jwks.json" {
					t.Errorf("unexpected body: %v", body)
				}
				if identifier == "custom-client" {
					if body["identifier"] != wantID {
						t.Errorf("identifier = %v", body["identifier"])
					}
				} else if _, ok := body["identifier"]; ok {
					t.Error("identifier must be omitted")
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{"id": "c1", "zone_id": "z1", "application_id": "a1", "type": "public-key", "jwks_uri": "https://example.com/jwks.json", "identifier": wantID}); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			r := &ApplicationPublicKeyCredentialResource{client: testClientFor(t, srv)}
			state := publicKeyCredentialState(t, r, identifier)
			resp := resource.CreateResponse{State: state}
			r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan(state)}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var data ApplicationPublicKeyCredentialModel
			if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
				t.Fatal(diags)
			}
			if data.Identifier.ValueString() != wantID || data.ID.ValueString() != "c1" || data.JWKSURI.ValueString() != "https://example.com/jwks.json" {
				t.Errorf("state = %+v", data)
			}
		})
	}
}

func TestApplicationPublicKeyCredentialRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/zones/z1/application-credentials/c1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"id": "c1", "zone_id": "z1", "application_id": "a1", "type": "public-key", "identifier": "remote-client", "jwks_uri": "https://example.com/remote.json"}); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()
	r := &ApplicationPublicKeyCredentialResource{client: testClientFor(t, srv)}
	state := publicKeyCredentialState(t, r, "old-client")
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var data ApplicationPublicKeyCredentialModel
	if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
		t.Fatal(diags)
	}
	if data.Identifier.ValueString() != "remote-client" || data.JWKSURI.ValueString() != "https://example.com/remote.json" {
		t.Errorf("state = %+v", data)
	}
}

func TestApplicationPublicKeyCredentialResponses(t *testing.T) {
	for _, operation := range []string{"create", "read", "delete"} {
		for _, status := range []int{200, 204, 400, 404} {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(status) }))
				defer srv.Close()
				r := &ApplicationPublicKeyCredentialResource{client: testClientFor(t, srv)}
				state := publicKeyCredentialState(t, r, "client")
				switch operation {
				case "create":
					resp := resource.CreateResponse{State: state}
					r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan(state)}, &resp)
					if !resp.Diagnostics.HasError() {
						t.Fatal("expected diagnostic for empty or error response")
					}
				case "read":
					resp := resource.ReadResponse{State: state}
					r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
					if status == 404 {
						if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
							t.Fatal("404 must remove resource")
						}
					} else if !resp.Diagnostics.HasError() {
						t.Fatal("expected diagnostic")
					}
				case "delete":
					resp := resource.DeleteResponse{State: state}
					r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
					if resp.Diagnostics.HasError() != (status != 204 && status != 404) {
						t.Fatalf("diagnostics = %v", resp.Diagnostics)
					}
				}
			})
		}
	}
}

func TestApplicationPublicKeyCredentialImport(t *testing.T) {
	r := &ApplicationPublicKeyCredentialResource{}
	for _, id := range []string{"zones/z1/application-credentials/c1", "", "z1/c1", "zones//application-credentials/c1", "zones/z1/application-credentials/"} {
		t.Run(id, func(t *testing.T) {
			resp := resource.ImportStateResponse{State: publicKeyCredentialState(t, r, nil)}
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
			valid := id == "zones/z1/application-credentials/c1"
			if resp.Diagnostics.HasError() == valid {
				t.Fatalf("diagnostics = %v", resp.Diagnostics)
			}
			if valid {
				var data ApplicationPublicKeyCredentialModel
				if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
					t.Fatal(diags)
				}
				if data.ID.ValueString() != "c1" || data.ZoneID.ValueString() != "z1" {
					t.Errorf("state = %+v", data)
				}
			}
		})
	}
}
