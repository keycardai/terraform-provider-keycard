package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// syncOAuth2WithIdentifierModifier keeps oauth2.issuer in sync with the
// identifier for providers that don't have an explicit oauth2 block.
//
// When the user only sets identifier (no oauth2 block), identifier and
// issuer should always match. On updates, if the user changes the
// identifier, this modifier ensures the issuer in the plan follows.
//
// When the user sets oauth2 explicitly, this modifier does nothing —
// identifier and issuer are independent.
type syncOAuth2WithIdentifierModifier struct{}

func (m syncOAuth2WithIdentifierModifier) Description(_ context.Context) string {
	return "Keeps oauth2.issuer in sync with identifier when oauth2 is not explicitly configured."
}

func (m syncOAuth2WithIdentifierModifier) MarkdownDescription(_ context.Context) string {
	return "Keeps `oauth2.issuer` in sync with `identifier` when `oauth2` is not explicitly configured."
}

func (m syncOAuth2WithIdentifierModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	// Resource is being created — nothing to carry forward.
	if req.State.Raw.IsNull() {
		return
	}

	// The user set oauth2 in config — they control issuer directly.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// Config uses interpolation — leave unknown to avoid breaking references.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Carry forward the previous oauth2 value (same as UseStateForUnknown).
	resp.PlanValue = req.StateValue

	// If there's no previous oauth2 (non-OAuth2 provider), nothing to sync.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// The user didn't set oauth2 in config, so this is an identifier-only
	// provider where identifier and issuer should match. Read identifier
	// from config (not plan) because plan modifier execution order across
	// attributes is non-deterministic.
	var identifierConfig types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("identifier"), &identifierConfig)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Identifier not in config either — nothing to sync.
	if identifierConfig.IsNull() || identifierConfig.IsUnknown() {
		return
	}

	var stateOAuth2 OAuth2ProviderModel
	resp.Diagnostics.Append(req.StateValue.As(ctx, &stateOAuth2, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Issuer already matches — no update needed.
	if stateOAuth2.Issuer.ValueString() == identifierConfig.ValueString() {
		return
	}

	// Identifier changed — update issuer to match.
	stateOAuth2.Issuer = identifierConfig

	var diags diag.Diagnostics
	resp.PlanValue, diags = types.ObjectValueFrom(ctx, stateOAuth2.AttributeTypes(), stateOAuth2)
	resp.Diagnostics.Append(diags...)
}
