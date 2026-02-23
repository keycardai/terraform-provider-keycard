package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// defaultIdentifierFromIssuerModifier defaults the identifier to the
// oauth2.issuer value when not explicitly configured. On updates, if the
// user removes identifier from config, it reverts to the issuer value.
// When oauth2 is not configured (e.g. vault providers), identifier must
// be set explicitly — ValidateConfig enforces this.
type defaultIdentifierFromIssuerModifier struct{}

func (m defaultIdentifierFromIssuerModifier) Description(_ context.Context) string {
	return "Defaults to the oauth2.issuer value when not configured."
}

func (m defaultIdentifierFromIssuerModifier) MarkdownDescription(_ context.Context) string {
	return "Defaults to the `oauth2.issuer` value when not configured."
}

func (m defaultIdentifierFromIssuerModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the user explicitly set identifier in config, keep it.
	if !req.ConfigValue.IsNull() {
		return
	}

	// User did not set identifier — try to default from oauth2.issuer.
	// Read from config (not plan) to get the user-specified oauth2 block.
	var oauth2Config types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("oauth2"), &oauth2Config)...)
	if resp.Diagnostics.HasError() || oauth2Config.IsNull() || oauth2Config.IsUnknown() {
		// oauth2 not configured — identifier stays unknown.
		// ValidateConfig will error if both are missing.
		return
	}

	var oauth2Model OAuth2ProviderModel
	resp.Diagnostics.Append(oauth2Config.As(ctx, &oauth2Model, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !oauth2Model.Issuer.IsNull() && !oauth2Model.Issuer.IsUnknown() {
		resp.PlanValue = oauth2Model.Issuer
	}
}
