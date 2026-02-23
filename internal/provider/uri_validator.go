package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// isValidURIValidator validates that a string value is a valid URI.
type isValidURIValidator struct{}

func (v isValidURIValidator) Description(ctx context.Context) string {
	return "value must be a valid URI"
}

func (v isValidURIValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid URI"
}

func (v isValidURIValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URI",
			fmt.Sprintf("Expected a valid URI with scheme and host, got: %s", value),
		)
	}
}

// IsValidURI returns a validator that checks whether a string is a valid URI.
func IsValidURI() validator.String {
	return isValidURIValidator{}
}
