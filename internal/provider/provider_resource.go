package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProviderResource{}
var _ resource.ResourceWithImportState = &ProviderResource{}

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
	AuthorizationEndpoint types.String `tfsdk:"authorization_endpoint"`
	TokenEndpoint         types.String `tfsdk:"token_endpoint"`
}

func (m OAuth2ProviderModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"authorization_endpoint": types.StringType,
		"token_endpoint":         types.StringType,
	}
}

func (r *ProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provider"
}

func (r *ProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Keycard provider. A provider is a system that supplies access to resources and allows actors (users or applications) to authenticate.",

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
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "User-specified identifier, must be unique within the zone.",
				Required:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth 2.0 client identifier.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth 2.0 client secret.",
				Optional:            true,
				Sensitive:           true,
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth 2.0 protocol configuration.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"authorization_endpoint": schema.StringAttribute{
						MarkdownDescription: "OAuth 2.0 Authorization endpoint URL.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"token_endpoint": schema.StringAttribute{
						MarkdownDescription: "OAuth 2.0 Token endpoint URL.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
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

		// Only set protocols if at least one endpoint is provided
		if (!oauth2Data.AuthorizationEndpoint.IsNull() && !oauth2Data.AuthorizationEndpoint.IsUnknown()) ||
			(!oauth2Data.TokenEndpoint.IsNull() && !oauth2Data.TokenEndpoint.IsUnknown()) {

			createReq.Protocols = &client.ProviderProtocolCreate{
				Oauth2: &client.ProviderOAuth2ProtocolCreate{},
			}
			if !oauth2Data.AuthorizationEndpoint.IsNull() && !oauth2Data.AuthorizationEndpoint.IsUnknown() {
				createReq.Protocols.Oauth2.AuthorizationEndpoint = oauth2Data.AuthorizationEndpoint.ValueStringPointer()
			}

			if !oauth2Data.TokenEndpoint.IsNull() && !oauth2Data.TokenEndpoint.IsUnknown() {
				createReq.Protocols.Oauth2.TokenEndpoint = oauth2Data.TokenEndpoint.ValueStringPointer()
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

	// Set protocols.oauth2 fields
	// Handle both null (to clear) and non-null (to set) values
	if !data.OAuth2.IsUnknown() {
		if data.OAuth2.IsNull() {
			// Explicitly clear protocols to allow server defaults
			updateReq.Protocols = nullable.NewNullNullable[client.ProviderProtocolUpdate]()
		} else {
			var oauth2Data OAuth2ProviderModel
			diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// Only set protocols if at least one endpoint is provided
			if !oauth2Data.AuthorizationEndpoint.IsUnknown() || !oauth2Data.TokenEndpoint.IsUnknown() {
				oauth2Update := client.ProviderOAuth2ProtocolUpdate{}
				if !oauth2Data.AuthorizationEndpoint.IsNull() && !oauth2Data.AuthorizationEndpoint.IsUnknown() {
					oauth2Update.AuthorizationEndpoint = StringValueNullable(oauth2Data.AuthorizationEndpoint)
				}

				if !oauth2Data.TokenEndpoint.IsNull() && !oauth2Data.TokenEndpoint.IsUnknown() {
					oauth2Update.TokenEndpoint = StringValueNullable(oauth2Data.TokenEndpoint)
				}
				protocolUpdate := client.ProviderProtocolUpdate{
					Oauth2: nullable.NewNullableWithValue(oauth2Update),
				}
				updateReq.Protocols = nullable.NewNullableWithValue(protocolUpdate)
			}
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
