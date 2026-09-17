package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// userConfig builds a data source config with the given string attributes set
// and every other attribute null.
func userConfig(t *testing.T, d *UserDataSource, attrs map[string]string) tfsdk.Config {
	t.Helper()
	var schemaResp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
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
	for name, v := range attrs {
		vals[name] = tftypes.NewValue(tftypes.String, v)
	}
	return tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
}

func validateUserConfig(t *testing.T, attrs map[string]string) bool {
	t.Helper()
	d := &UserDataSource{}
	cfg := userConfig(t, d, attrs)
	ctx := context.Background()
	// ValidateDataSource assigns rather than appends, so use one response per validator.
	for _, v := range d.ConfigValidators(ctx) {
		resp := &datasource.ValidateConfigResponse{}
		v.ValidateDataSource(ctx, datasource.ValidateConfigRequest{Config: cfg}, resp)
		if resp.Diagnostics.HasError() {
			return true
		}
	}
	// Attribute-level string validators.
	for name, attr := range cfg.Schema.GetAttributes() {
		strAttr, ok := attr.(schema.StringAttribute)
		if !ok {
			continue
		}
		var val types.String
		if diags := cfg.GetAttribute(ctx, path.Root(name), &val); diags.HasError() {
			t.Fatalf("get %s: %s", name, diags)
		}
		for _, v := range strAttr.Validators {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root(name), Config: cfg, ConfigValue: val}, resp)
			if resp.Diagnostics.HasError() {
				return true
			}
		}
	}
	return false
}

func TestUserDataSourceValidators(t *testing.T) {
	cases := map[string]struct {
		attrs   map[string]string
		wantErr bool
	}{
		"id":                     {map[string]string{"zone_id": "z", "id": "u"}, false},
		"identifier":             {map[string]string{"zone_id": "z", "identifier": "alice"}, false},
		"email+issuer":           {map[string]string{"zone_id": "z", "email": "a@x.io", "issuer": "https://idp"}, false},
		"subject+issuer":         {map[string]string{"zone_id": "z", "subject": "s", "issuer": "https://idp"}, false},
		"none":                   {map[string]string{"zone_id": "z"}, true},
		"email without issuer":   {map[string]string{"zone_id": "z", "email": "a@x.io"}, false},
		"subject without issuer": {map[string]string{"zone_id": "z", "subject": "s"}, true},
		"issuer alone":           {map[string]string{"zone_id": "z", "issuer": "https://idp"}, true},
		"id and identifier":      {map[string]string{"zone_id": "z", "id": "u", "identifier": "alice"}, true},
		"id with issuer":         {map[string]string{"zone_id": "z", "id": "u", "issuer": "https://idp"}, true},
		"identifier with issuer": {map[string]string{"zone_id": "z", "identifier": "alice", "issuer": "https://idp"}, true},
		"email and subject":      {map[string]string{"zone_id": "z", "email": "a@x.io", "subject": "s", "issuer": "https://idp"}, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := validateUserConfig(t, tc.attrs); got != tc.wantErr {
				t.Errorf("hasError = %v, want %v", got, tc.wantErr)
			}
		})
	}
}

func userItem(id, email string) map[string]any {
	return map[string]any{
		"id": id, "zone_id": "z1", "identifier": id, "email": email, "status": "active",
		"external": true, "subject": "sub-" + id, "issuer": "https://idp.example.com", "provider_id": "p1",
	}
}

// userServer serves GET /zones/{z}/users/{id} from byID and GET /zones/{z}/users from list,
// recording the last list query.
func userServer(t *testing.T, byID map[string]any, list []map[string]any, lastQuery *url.Values) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body any
		if strings.HasSuffix(req.URL.Path, "/users") {
			if lastQuery != nil {
				*lastQuery = req.URL.Query()
			}
			body = map[string]any{"items": list, "pagination": map[string]any{}}
		} else {
			if byID == nil {
				w.WriteHeader(http.StatusNotFound)
				body = map[string]any{"error": "not_found"}
			} else {
				body = byID
			}
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Errorf("encoding response: %s", err)
		}
	}))
}

func readUser(t *testing.T, srv *httptest.Server, attrs map[string]string) (UserModel, datasource.ReadResponse) {
	t.Helper()
	d := &UserDataSource{client: testClientFor(t, srv)}
	cfg := userConfig(t, d, attrs)
	resp := datasource.ReadResponse{State: tfsdk.State(cfg)}
	d.Read(context.Background(), datasource.ReadRequest{Config: cfg}, &resp)
	var data UserModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
			t.Fatalf("state: %s", diags)
		}
	}
	return data, resp
}

func TestUserDataSourceRead_byID(t *testing.T) {
	srv := userServer(t, userItem("u1", "alice@example.com"), nil, nil)
	defer srv.Close()

	data, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "id": "u1"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}
	checks := map[string]string{
		"id": data.ID.ValueString(), "identifier": data.Identifier.ValueString(), "email": data.Email.ValueString(),
		"status": data.Status.ValueString(), "subject": data.Subject.ValueString(), "issuer": data.Issuer.ValueString(),
		"provider_id": data.ProviderID.ValueString(),
	}
	want := map[string]string{
		"id": "u1", "identifier": "u1", "email": "alice@example.com", "status": "active",
		"subject": "sub-u1", "issuer": "https://idp.example.com", "provider_id": "p1",
	}
	for k, w := range want {
		if checks[k] != w {
			t.Errorf("%s = %q, want %q", k, checks[k], w)
		}
	}
	if !data.External.ValueBool() {
		t.Error("external = false, want true")
	}
}

func TestUserDataSourceRead_byIDNotFound(t *testing.T) {
	srv := userServer(t, nil, nil, nil)
	defer srv.Close()

	_, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "id": "missing"})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected not-found error")
	}
	if got := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(got, "missing") {
		t.Errorf("error should name the id, got: %s", got)
	}
}

func TestUserDataSourceRead_byEmailAndIssuer(t *testing.T) {
	var q url.Values
	srv := userServer(t, nil, []map[string]any{userItem("u1", "alice@example.com")}, &q)
	defer srv.Close()

	data, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "email": "alice@example.com", "issuer": "https://idp.example.com"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}
	if data.ID.ValueString() != "u1" {
		t.Errorf("id = %q, want u1", data.ID.ValueString())
	}
	if got := q.Get("filter[email]"); got != "alice@example.com" {
		t.Errorf("filter[email] = %q", got)
	}
	if got := q.Get("filter[issuer]"); got != "https://idp.example.com" {
		t.Errorf("filter[issuer] = %q", got)
	}
	if q.Has("filter[subject]") || q.Has("filter[identifier]") {
		t.Errorf("unexpected filters in query: %v", q)
	}
	if got := q.Get("limit"); got != "2" {
		t.Errorf("limit = %q, want 2", got)
	}
}

func TestUserDataSourceRead_bySubjectAndIssuer(t *testing.T) {
	var q url.Values
	srv := userServer(t, nil, []map[string]any{userItem("u1", "alice@example.com")}, &q)
	defer srv.Close()

	_, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "subject": "sub-u1", "issuer": "https://idp.example.com"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}
	if got := q.Get("filter[subject]"); got != "sub-u1" {
		t.Errorf("filter[subject] = %q", got)
	}
	if q.Has("filter[email]") {
		t.Errorf("unexpected filter[email] in query: %v", q)
	}
}

func TestUserDataSourceRead_byIdentifier(t *testing.T) {
	var q url.Values
	srv := userServer(t, nil, []map[string]any{userItem("u1", "alice@example.com")}, &q)
	defer srv.Close()

	_, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "identifier": "u1"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}
	if got := q.Get("filter[identifier]"); got != "u1" {
		t.Errorf("filter[identifier] = %q", got)
	}
	if q.Has("filter[issuer]") {
		t.Errorf("unexpected filter[issuer] in query: %v", q)
	}
}

func TestUserDataSourceRead_zeroMatches(t *testing.T) {
	srv := userServer(t, nil, []map[string]any{}, nil)
	defer srv.Close()

	_, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "email": "nobody@example.com", "issuer": "https://idp.example.com"})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected not-found error")
	}
	got := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(got, "nobody@example.com") || !strings.Contains(got, "https://idp.example.com") {
		t.Errorf("error should name the lookup values, got: %s", got)
	}
}

func TestUserDataSourceRead_byEmailOnly(t *testing.T) {
	var q url.Values
	srv := userServer(t, nil, []map[string]any{userItem("u1", "alice@example.com")}, &q)
	defer srv.Close()

	data, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "email": "alice@example.com"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
	}
	if data.ID.ValueString() != "u1" {
		t.Errorf("id = %q, want u1", data.ID.ValueString())
	}
	if q.Has("filter[issuer]") {
		t.Errorf("unexpected filter[issuer] in query: %v", q)
	}
}

func TestUserDataSourceRead_multipleMatchesByEmailSuggestsIssuer(t *testing.T) {
	srv := userServer(t, nil, []map[string]any{userItem("u1", "a@x.io"), userItem("u2", "a@x.io")}, nil)
	defer srv.Close()

	_, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "email": "a@x.io"})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected ambiguity error")
	}
	got := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(got, "u1") || !strings.Contains(got, "u2") {
		t.Errorf("error should list every matched id, got: %s", got)
	}
	if !strings.Contains(got, "issuer") {
		t.Errorf("error should tell the caller to add issuer, got: %s", got)
	}
}

func TestUserDataSourceRead_multipleMatchesWithIssuerSuggestsID(t *testing.T) {
	srv := userServer(t, nil, []map[string]any{userItem("u1", "a@x.io"), userItem("u2", "a@x.io")}, nil)
	defer srv.Close()

	_, resp := readUser(t, srv, map[string]string{"zone_id": "z1", "email": "a@x.io", "issuer": "https://idp.example.com"})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected ambiguity error")
	}
	got := resp.Diagnostics.Errors()[0].Detail()
	if strings.Contains(got, "add `issuer`") || !strings.Contains(got, "`id`") {
		t.Errorf("error should tell the caller to use id, got: %s", got)
	}
}
