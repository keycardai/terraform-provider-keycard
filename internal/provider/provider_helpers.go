package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

// updateProviderModelFromAPIResponse updates the model with data from a provider API response.
// It handles mapping of all fields except client_secret, which should be preserved from plan/state.
func updateProviderModelFromAPIResponse(ctx context.Context, provider *client.Provider, data *ProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map basic fields
	data.ID = types.StringValue(provider.Id)
	data.Name = types.StringValue(provider.Name)
	data.Description = NullableStringValue(provider.Description)
	data.Identifier = types.StringValue(provider.Identifier)
	data.ClientID = NullableStringValue(provider.ClientId)

	// Note: client_secret is not updated here as it's write-only in the API
	// It should already be set from plan/state in the calling method
	protocols, err := provider.Protocols.Get()
	if err == nil {
		oauth2, err := protocols.Oauth2.Get()
		if err == nil {
			oauth2Model := OAuth2ProviderModel{
				AuthorizationEndpoint: NullableStringValue(oauth2.AuthorizationEndpoint),
				TokenEndpoint:         NullableStringValue(oauth2.TokenEndpoint),
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
	data.Description = NullableStringValue(provider.Description)
	data.Identifier = types.StringValue(provider.Identifier)
	data.ClientID = NullableStringValue(provider.ClientId)

	// Map protocols.oauth2 fields if present
	protocols, err := provider.Protocols.Get()
	if err == nil {
		oauth2, err := protocols.Oauth2.Get()
		if err == nil {
			oauth2Model := OAuth2ProviderModel{
				AuthorizationEndpoint: NullableStringValue(oauth2.AuthorizationEndpoint),
				TokenEndpoint:         NullableStringValue(oauth2.TokenEndpoint),
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
	} else {
		data.OAuth2 = types.ObjectNull(OAuth2ProviderModel{}.AttributeTypes())
	}

	return diags
}

func NullableStringValue(val nullable.Nullable[string]) basetypes.StringValue {
	str, err := val.Get()
	if err != nil {
		return types.StringNull()
	}

	return types.StringValue(str)
}

func StringValueNullable(val basetypes.StringValue) nullable.Nullable[string] {
	switch {
	case val.IsNull():
		return nullable.NewNullNullable[string]()
	case val.IsUnknown():
		return nullable.Nullable[string]{}
	default:
		return nullable.NewNullableWithValue(val.ValueString())
	}
}

func BoolValueNullable(val basetypes.BoolValue) nullable.Nullable[bool] {
	switch {
	case val.IsNull():
		return nullable.NewNullNullable[bool]()
	case val.IsUnknown():
		return nullable.Nullable[bool]{}
	default:
		return nullable.NewNullableWithValue(val.ValueBool())
	}
}
