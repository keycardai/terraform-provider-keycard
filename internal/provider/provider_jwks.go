package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

// jwksFromTerraform decodes a JWK Set from its Terraform JSON string into the
// API type. Callers must only pass a known, non-null value.
func jwksFromTerraform(val jsontypes.Normalized) (client.ProviderJwks, diag.Diagnostics) {
	var diags diag.Diagnostics
	var jwks client.ProviderJwks
	if err := json.Unmarshal([]byte(val.ValueString()), &jwks); err != nil {
		diags.AddAttributeError(
			path.Root("oauth2").AtName("jwks"),
			"Invalid JWKS",
			fmt.Sprintf("Unable to decode the JWK Set: %s", err),
		)
	}
	return jwks, diags
}

// jwksToTerraform encodes the API's JWK Set as compact JSON with sorted
// members, the same form Terraform's jsonencode produces. Absent or null
// becomes a null value.
func jwksToTerraform(val nullable.Nullable[client.ProviderJwks]) (jsontypes.Normalized, diag.Diagnostics) {
	var diags diag.Diagnostics
	jwks, err := val.Get()
	if err != nil {
		return jsontypes.NewNormalizedNull(), diags
	}

	encoded, err := json.Marshal(jwks)
	if err != nil {
		diags.AddError("Invalid JWKS", fmt.Sprintf("Unable to encode the JWK Set returned by the API: %s", err))
		return jsontypes.NewNormalizedNull(), diags
	}
	return jsontypes.NewNormalizedValue(string(encoded)), diags
}

// jwksUpdateValue returns the jwks value for a PATCH request. A configured
// value is sent as is. A value removed from config is sent as null so the API
// clears it. A value that was never set is omitted, so the request doesn't
// name a field the provider never managed.
func jwksUpdateValue(planned, prior jsontypes.Normalized) (nullable.Nullable[client.ProviderJwks], diag.Diagnostics) {
	switch {
	case planned.IsUnknown():
		return nullable.Nullable[client.ProviderJwks]{}, nil
	case planned.IsNull():
		if prior.IsNull() || prior.IsUnknown() {
			return nullable.Nullable[client.ProviderJwks]{}, nil
		}
		return nullable.NewNullNullable[client.ProviderJwks](), nil
	default:
		jwks, diags := jwksFromTerraform(planned)
		return nullable.NewNullableWithValue(jwks), diags
	}
}

// jwksSetValidator checks that a value is a JWK Set: a JSON object whose only
// member is a non-empty "keys" array of objects. The API validates the keys
// themselves. The jsontypes.Normalized type reports invalid JSON.
type jwksSetValidator struct{}

func (v jwksSetValidator) Description(_ context.Context) string {
	return `Value must be a JWK Set: a JSON object with a non-empty "keys" array.`
}

func (v jwksSetValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jwksSetValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	raw := []byte(req.ConfigValue.ValueString())
	if !json.Valid(raw) {
		// Reported by the jsontypes.Normalized type.
		return
	}

	fail := func(detail string) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JWKS", detail)
	}

	var set map[string]json.RawMessage
	if err := json.Unmarshal(raw, &set); err != nil || set == nil {
		fail(`The value must be a JWK Set, a JSON object such as {"keys": [...]}. A bare array or a single key is not accepted.`)
		return
	}

	for member := range set {
		if member != "keys" {
			fail(fmt.Sprintf(`The JWK Set may only contain a "keys" member, but it also contains %q.`, member))
			return
		}
	}

	keysRaw, ok := set["keys"]
	if !ok {
		fail(`The JWK Set must contain a "keys" array.`)
		return
	}

	var keys []map[string]json.RawMessage
	if err := json.Unmarshal(keysRaw, &keys); err != nil {
		fail(`The "keys" member must be an array of JSON objects.`)
		return
	}
	if len(keys) == 0 {
		fail(`The "keys" array must contain at least one key.`)
		return
	}
	for i, key := range keys {
		if key == nil {
			fail(fmt.Sprintf(`keys[%d] must be a JSON object.`, i))
			return
		}
	}
}

// jwksSemanticEqualityModifier keeps the prior state value when the configured
// JWK Set has the same content in a different format, such as other
// whitespace or member order, so reformatting doesn't plan an update.
type jwksSemanticEqualityModifier struct{}

func (m jwksSemanticEqualityModifier) Description(_ context.Context) string {
	return "Keeps the prior value when the configured JSON is semantically equal to it."
}

func (m jwksSemanticEqualityModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m jwksSemanticEqualityModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if req.ConfigValue.Equal(req.StateValue) {
		return
	}

	equal, diags := jsontypes.NewNormalizedValue(req.ConfigValue.ValueString()).
		StringSemanticEquals(ctx, jsontypes.NewNormalizedValue(req.StateValue.ValueString()))
	resp.Diagnostics.Append(diags...)
	if equal {
		resp.PlanValue = req.StateValue
	}
}

// clearableStringUpdateValue returns a string value for a PATCH request with
// the same rules as jwksUpdateValue: send a configured value, send null for a
// value removed from config, and omit a value that was never set.
func clearableStringUpdateValue(planned, prior basetypes.StringValue) nullable.Nullable[string] {
	switch {
	case planned.IsUnknown():
		return nullable.Nullable[string]{}
	case planned.IsNull():
		if prior.IsNull() || prior.IsUnknown() {
			return nullable.Nullable[string]{}
		}
		return nullable.NewNullNullable[string]()
	default:
		return nullable.NewNullableWithValue(planned.ValueString())
	}
}
