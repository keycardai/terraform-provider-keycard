package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                   = &ProviderResource{}
	_ resource.ResourceWithImportState    = &ProviderResource{}
	_ resource.ResourceWithUpgradeState   = &ProviderResource{}
	_ resource.ResourceWithValidateConfig = &ProviderResource{}
)

func NewProviderResource() resource.Resource {
	return &ProviderResource{}
}

// ProviderResource defines the resource implementation.
type ProviderResource struct {
	client *client.ClientWithResponses
}

// ProviderResourceModel describes the resource data model.
type ProviderResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ZoneID       types.String `tfsdk:"zone_id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Identifier   types.String `tfsdk:"identifier"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	OAuth2       types.Object `tfsdk:"oauth2"`
}

// OAuth2ProviderModel describes the nested oauth2 block data model.
type OAuth2ProviderModel struct {
	Issuer                  types.String `tfsdk:"issuer"`
	AuthorizationEndpoint   types.String `tfsdk:"authorization_endpoint"`
	TokenEndpoint           types.String `tfsdk:"token_endpoint"`
	AuthorizationParameters types.Map    `tfsdk:"authorization_parameters"`
}

func (m OAuth2ProviderModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"issuer":                   types.StringType,
		"authorization_endpoint":   types.StringType,
		"token_endpoint":           types.StringType,
		"authorization_parameters": types.MapType{ElemType: types.StringType},
	}
}

func (r *ProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provider"
}

func (r *ProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Keycard provider. A provider is a system that supplies access to resources and allows actors (users or applications) to authenticate.",
		Version:             2,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this provider belongs to. Changing this will replace the provider.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the provider.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the provider's purpose.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "User-specified identifier, must be unique within the zone. Defaults to the `oauth2.issuer` value when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					defaultIdentifierFromIssuerModifier{},
				},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth 2.0 client identifier.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth 2.0 client secret.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth 2.0 protocol configuration. When provided, `issuer` is required. If `identifier` is not set, it defaults to the `issuer` value.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"issuer": schema.StringAttribute{
						MarkdownDescription: "OIDC issuer URL used for discovery and token validation. Must be a valid URI.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
							IsValidURI(),
						},
					},
					"authorization_endpoint": schema.StringAttribute{
						MarkdownDescription: "OAuth 2.0 Authorization endpoint URL.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"token_endpoint": schema.StringAttribute{
						MarkdownDescription: "OAuth 2.0 Token endpoint URL.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"authorization_parameters": schema.MapAttribute{
						MarkdownDescription: "Custom query parameters appended to authorization redirect URLs. Use for non-standard providers that require extra parameters (e.g. Google: prompt=consent, access_type=offline).",
						Optional:            true,
						ElementType:         types.StringType,
					},
				},
				PlanModifiers: []planmodifier.Object{
					syncOAuth2WithIdentifierModifier{},
				},
			},
		},
	}
}

func (r *ProviderResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var identifier types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("identifier"), &identifier)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var oauth2 types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("oauth2"), &oauth2)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if identifier.IsNull() && oauth2.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Configuration",
			"At least one of \"identifier\" or \"oauth2\" must be configured. "+
				"When \"oauth2\" is provided, \"identifier\" defaults to the issuer value. "+
				"When \"oauth2\" is not provided, \"identifier\" is required.",
		)
	}
}

func (r *ProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProviderResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the create request
	createReq := client.ProviderCreate{
		Name:       data.Name.ValueString(),
		Identifier: data.Identifier.ValueString(),
	}

	// Set description if provided
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		createReq.Description = StringValueNullable(data.Description)
	}

	// Set client_id if provided
	if !data.ClientID.IsNull() && !data.ClientID.IsUnknown() {
		clientID := data.ClientID.ValueString()
		createReq.ClientId = &clientID
	}

	// Set client_secret if provided
	if !data.ClientSecret.IsNull() && !data.ClientSecret.IsUnknown() {
		clientSecret := data.ClientSecret.ValueString()
		createReq.ClientSecret = &clientSecret
	}

	// Set protocols.oauth2 fields if oauth2 block is provided
	if !data.OAuth2.IsNull() && !data.OAuth2.IsUnknown() {
		var oauth2Data OAuth2ProviderModel
		diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		createReq.Protocols = &client.ProviderProtocolCreate{
			Oauth2: &client.ProviderOAuth2ProtocolCreate{
				Issuer: oauth2Data.Issuer.ValueStringPointer(),
			},
		}

		if !oauth2Data.AuthorizationEndpoint.IsNull() && !oauth2Data.AuthorizationEndpoint.IsUnknown() {
			createReq.Protocols.Oauth2.AuthorizationEndpoint = oauth2Data.AuthorizationEndpoint.ValueStringPointer()
		}

		if !oauth2Data.TokenEndpoint.IsNull() && !oauth2Data.TokenEndpoint.IsUnknown() {
			createReq.Protocols.Oauth2.TokenEndpoint = oauth2Data.TokenEndpoint.ValueStringPointer()
		}

		if !oauth2Data.AuthorizationParameters.IsNull() && !oauth2Data.AuthorizationParameters.IsUnknown() {
			params := make(map[string]string)
			diags := oauth2Data.AuthorizationParameters.ElementsAs(ctx, &params, false)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				createReq.Protocols.Oauth2.AuthorizationParameters = &params
			}
		}
	}

	// Create the provider
	createResp, err := r.client.CreateProviderWithResponse(ctx, data.ZoneID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create provider, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create provider, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create provider, no response body")
		return
	}

	// Update the model with the response data
	provider := createResp.JSON200
	resp.Diagnostics.Append(updateProviderModelFromAPIResponse(ctx, provider, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProviderResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the provider
	getResp, err := r.client.GetProviderWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read provider, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Provider was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read provider, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read provider, no response body")
		return
	}

	// Update the model with the response data
	provider := getResp.JSON200
	resp.Diagnostics.Append(updateProviderModelFromAPIResponse(ctx, provider, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProviderResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the update request
	updateReq := client.ProviderUpdate{}

	// Set name if changed
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		updateReq.Name = &name
	}

	// Set description (including null to remove it)
	if !data.Description.IsUnknown() {
		updateReq.Description = StringValueNullable(data.Description)
	}

	// Set identifier at root level
	if !data.Identifier.IsNull() && !data.Identifier.IsUnknown() {
		identifier := data.Identifier.ValueString()
		updateReq.Identifier = &identifier
	}

	// Set client_id at root level
	if !data.ClientID.IsUnknown() {
		updateReq.ClientId = StringValueNullable(data.ClientID)
	}

	// Set client_secret at root level
	if !data.ClientSecret.IsUnknown() {
		updateReq.ClientSecret = StringValueNullable(data.ClientSecret)
	}

	// Set protocols.oauth2 fields if oauth2 block is provided
	if !data.OAuth2.IsUnknown() {
		if data.OAuth2.IsNull() {
			// Null means this is a non-OAuth2 provider (e.g. vault) that has
			// no oauth2 in state. Don't send protocols.
			//
			// Identifier-only providers always have oauth2 in state (the API
			// copies identifier into issuer on create), so the plan modifier
			// gives them a non-null value and they go through the else branch.
		} else {
			var oauth2Data OAuth2ProviderModel
			diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			oauth2Update := client.ProviderOAuth2ProtocolUpdate{
				// Always send issuer explicitly — it cannot be null once stored
				Issuer: oauth2Data.Issuer.ValueStringPointer(),
			}

			if !oauth2Data.AuthorizationEndpoint.IsNull() && !oauth2Data.AuthorizationEndpoint.IsUnknown() {
				oauth2Update.AuthorizationEndpoint = StringValueNullable(oauth2Data.AuthorizationEndpoint)
			}

			if !oauth2Data.TokenEndpoint.IsNull() && !oauth2Data.TokenEndpoint.IsUnknown() {
				oauth2Update.TokenEndpoint = StringValueNullable(oauth2Data.TokenEndpoint)
			}

			if !oauth2Data.AuthorizationParameters.IsNull() && !oauth2Data.AuthorizationParameters.IsUnknown() {
				params := make(map[string]string)
				diags := oauth2Data.AuthorizationParameters.ElementsAs(ctx, &params, false)
				resp.Diagnostics.Append(diags...)
				if !resp.Diagnostics.HasError() {
					oauth2Update.AuthorizationParameters = nullable.NewNullableWithValue(params)
				}
			} else if oauth2Data.AuthorizationParameters.IsNull() {
				oauth2Update.AuthorizationParameters = nullable.NewNullNullable[map[string]string]()
			}

			protocolUpdate := client.ProviderProtocolUpdate{
				Oauth2: nullable.NewNullableWithValue(oauth2Update),
			}
			updateReq.Protocols = nullable.NewNullableWithValue(protocolUpdate)
		}
	}

	// Update the provider
	updateResp, err := r.client.UpdateProviderWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update provider, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update provider, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update provider, no response body")
		return
	}

	// Update the model with the response data
	provider := updateResp.JSON200
	resp.Diagnostics.Append(updateProviderModelFromAPIResponse(ctx, provider, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProviderResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the provider
	deleteResp, err := r.client.DeleteProviderWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete provider, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete provider, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *ProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/providers/{provider-id}
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 || parts[0] != "zones" || parts[2] != "providers" || parts[1] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zones/{zone-id}/providers/{provider-id}', got: %s", req.ID),
		)
		return
	}

	zoneID := parts[1]
	providerID := parts[3]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), providerID)...)
}

func (r *ProviderResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			// Migrate from schema version 0 (no issuer) to version 1 (issuer required).
			// Populates oauth2.issuer from the identifier value, matching the API's
			// backward-compatible default behavior.
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":            schema.StringAttribute{Computed: true},
					"zone_id":       schema.StringAttribute{Required: true},
					"name":          schema.StringAttribute{Required: true},
					"description":   schema.StringAttribute{Optional: true},
					"identifier":    schema.StringAttribute{Required: true},
					"client_id":     schema.StringAttribute{Optional: true},
					"client_secret": schema.StringAttribute{Optional: true, Sensitive: true},
					"oauth2": schema.SingleNestedAttribute{
						Optional: true,
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"authorization_endpoint": schema.StringAttribute{Optional: true, Computed: true},
							"token_endpoint":         schema.StringAttribute{Optional: true, Computed: true},
						},
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// Read v0 state
				type v0OAuth2Model struct {
					AuthorizationEndpoint types.String `tfsdk:"authorization_endpoint"`
					TokenEndpoint         types.String `tfsdk:"token_endpoint"`
				}
				type v0Model struct {
					ID           types.String `tfsdk:"id"`
					ZoneID       types.String `tfsdk:"zone_id"`
					Name         types.String `tfsdk:"name"`
					Description  types.String `tfsdk:"description"`
					Identifier   types.String `tfsdk:"identifier"`
					ClientID     types.String `tfsdk:"client_id"`
					ClientSecret types.String `tfsdk:"client_secret"`
					OAuth2       types.Object `tfsdk:"oauth2"`
				}

				var priorState v0Model
				resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Build the new oauth2 object with issuer populated from identifier.
				// This matches the API's backward-compatible default: when issuer was
				// not provided on create, identifier was copied into issuer.
				// For non-OAuth2 providers (e.g. vault), keep oauth2 null.
				var oauth2Obj basetypes.ObjectValue
				if !priorState.OAuth2.IsNull() && !priorState.OAuth2.IsUnknown() {
					var oauth2 v0OAuth2Model
					resp.Diagnostics.Append(priorState.OAuth2.As(ctx, &oauth2, basetypes.ObjectAsOptions{})...)
					if resp.Diagnostics.HasError() {
						return
					}

					newOAuth2 := OAuth2ProviderModel{
						Issuer:                  priorState.Identifier,
						AuthorizationEndpoint:   oauth2.AuthorizationEndpoint,
						TokenEndpoint:           oauth2.TokenEndpoint,
						AuthorizationParameters: types.MapNull(types.StringType),
					}
					var diags diag.Diagnostics
					oauth2Obj, diags = types.ObjectValueFrom(ctx, newOAuth2.AttributeTypes(), newOAuth2)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}
				} else {
					oauth2Obj = types.ObjectNull(OAuth2ProviderModel{}.AttributeTypes())
				}

				// Write upgraded state
				upgradedState := ProviderResourceModel{
					ID:           priorState.ID,
					ZoneID:       priorState.ZoneID,
					Name:         priorState.Name,
					Description:  priorState.Description,
					Identifier:   priorState.Identifier,
					ClientID:     priorState.ClientID,
					ClientSecret: priorState.ClientSecret,
					OAuth2:       oauth2Obj,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedState)...)
			},
		},
		1: {
			// Migrate from schema version 1 to version 2 (adds authorization_parameters).
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":            schema.StringAttribute{Computed: true},
					"zone_id":       schema.StringAttribute{Required: true},
					"name":          schema.StringAttribute{Required: true},
					"description":   schema.StringAttribute{Optional: true},
					"identifier":    schema.StringAttribute{Optional: true, Computed: true},
					"client_id":     schema.StringAttribute{Optional: true},
					"client_secret": schema.StringAttribute{Optional: true, Sensitive: true},
					"oauth2": schema.SingleNestedAttribute{
						Optional: true,
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"issuer":                 schema.StringAttribute{Required: true},
							"authorization_endpoint": schema.StringAttribute{Optional: true, Computed: true},
							"token_endpoint":         schema.StringAttribute{Optional: true, Computed: true},
						},
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				type v1OAuth2Model struct {
					Issuer                types.String `tfsdk:"issuer"`
					AuthorizationEndpoint types.String `tfsdk:"authorization_endpoint"`
					TokenEndpoint         types.String `tfsdk:"token_endpoint"`
				}
				type v1Model struct {
					ID           types.String `tfsdk:"id"`
					ZoneID       types.String `tfsdk:"zone_id"`
					Name         types.String `tfsdk:"name"`
					Description  types.String `tfsdk:"description"`
					Identifier   types.String `tfsdk:"identifier"`
					ClientID     types.String `tfsdk:"client_id"`
					ClientSecret types.String `tfsdk:"client_secret"`
					OAuth2       types.Object `tfsdk:"oauth2"`
				}

				var priorState v1Model
				resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
				if resp.Diagnostics.HasError() {
					return
				}

				var oauth2Obj basetypes.ObjectValue
				if !priorState.OAuth2.IsNull() && !priorState.OAuth2.IsUnknown() {
					var oauth2 v1OAuth2Model
					resp.Diagnostics.Append(priorState.OAuth2.As(ctx, &oauth2, basetypes.ObjectAsOptions{})...)
					if resp.Diagnostics.HasError() {
						return
					}

					newOAuth2 := OAuth2ProviderModel{
						Issuer:                  oauth2.Issuer,
						AuthorizationEndpoint:   oauth2.AuthorizationEndpoint,
						TokenEndpoint:           oauth2.TokenEndpoint,
						AuthorizationParameters: types.MapNull(types.StringType),
					}
					var diags diag.Diagnostics
					oauth2Obj, diags = types.ObjectValueFrom(ctx, newOAuth2.AttributeTypes(), newOAuth2)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}
				} else {
					oauth2Obj = types.ObjectNull(OAuth2ProviderModel{}.AttributeTypes())
				}

				upgradedState := ProviderResourceModel{
					ID:           priorState.ID,
					ZoneID:       priorState.ZoneID,
					Name:         priorState.Name,
					Description:  priorState.Description,
					Identifier:   priorState.Identifier,
					ClientID:     priorState.ClientID,
					ClientSecret: priorState.ClientSecret,
					OAuth2:       oauth2Obj,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedState)...)
			},
		},
	}
}
