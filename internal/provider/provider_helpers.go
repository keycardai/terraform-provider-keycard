package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// updateProviderModelFromAPIResponse updates the model with data from a provider API response.
// It handles mapping of all fields except client_secret, which should be preserved from plan/state.
func updateProviderModelFromAPIResponse(ctx context.Context, provider *client.Provider, data *ProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map basic fields
	data.ID = types.StringValue(provider.Id)
	data.Name = types.StringValue(provider.Name)
	data.Description = types.StringPointerValue(provider.Description)
	data.Identifier = types.StringValue(provider.Identifier)
	data.ClientID = types.StringPointerValue(provider.ClientId)

	// Note: client_secret is not updated here as it's write-only in the API
	// It should already be set from plan/state in the calling method

	// Map protocols.oauth2 fields if present
	if provider.Protocols != nil && provider.Protocols.Oauth2 != nil {
		oauth2Model := OAuth2ProviderModel{
			AuthorizationEndpoint: types.StringPointerValue(provider.Protocols.Oauth2.AuthorizationEndpoint),
			TokenEndpoint:         types.StringPointerValue(provider.Protocols.Oauth2.TokenEndpoint),
		}

		oauth2Obj, oauth2Diags := types.ObjectValueFrom(ctx, oauth2Model.AttributeTypes(), oauth2Model)
		diags.Append(oauth2Diags...)
		if diags.HasError() {
			return diags
		}
		data.OAuth2 = oauth2Obj
	} else {
		data.OAuth2 = types.ObjectNull(OAuth2ProviderModel{}.AttributeTypes())
	}

	return diags
}

// updateProviderDataSourceModelFromAPIResponse updates the data source model with data from a provider API response.
// This is used by the provider data source which doesn't include client_secret.
func updateProviderDataSourceModelFromAPIResponse(ctx context.Context, provider *client.Provider, data *ProviderDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map basic fields
	data.ID = types.StringValue(provider.Id)
	data.Name = types.StringValue(provider.Name)
	data.Description = types.StringPointerValue(provider.Description)
	data.Identifier = types.StringValue(provider.Identifier)
	data.ClientID = types.StringPointerValue(provider.ClientId)

	// Map protocols.oauth2 fields if present
	if provider.Protocols != nil && provider.Protocols.Oauth2 != nil {
		oauth2Model := OAuth2ProviderModel{
			AuthorizationEndpoint: types.StringPointerValue(provider.Protocols.Oauth2.AuthorizationEndpoint),
			TokenEndpoint:         types.StringPointerValue(provider.Protocols.Oauth2.TokenEndpoint),
		}

		oauth2Obj, oauth2Diags := types.ObjectValueFrom(ctx, oauth2Model.AttributeTypes(), oauth2Model)
		diags.Append(oauth2Diags...)
		if diags.HasError() {
			return diags
		}
		data.OAuth2 = oauth2Obj
	} else {
		data.OAuth2 = types.ObjectNull(OAuth2ProviderModel{}.AttributeTypes())
	}

	return diags
}
