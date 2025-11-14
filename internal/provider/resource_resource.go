package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ResourceResource{}
	_ resource.ResourceWithImportState = &ResourceResource{}
)

func NewResourceResource() resource.Resource {
	return &ResourceResource{}
}

// ResourceResource defines the resource implementation.
type ResourceResource struct {
	client *client.KeycardClient
}

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ZoneID               types.String `tfsdk:"zone_id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Identifier           types.String `tfsdk:"identifier"`
	CredentialProviderID types.String `tfsdk:"credential_provider_id"`
	ApplicationID        types.String `tfsdk:"application_id"`
	Metadata             types.Object `tfsdk:"metadata"`
	OAuth2               types.Object `tfsdk:"oauth2"`
}

// ResourceMetadataModel describes the nested metadata block data model.
type ResourceMetadataModel struct {
	DocsURL types.String `tfsdk:"docs_url"`
}

func (m ResourceMetadataModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"docs_url": types.StringType,
	}
}

// ResourceOAuth2Model describes the nested oauth2 block data model.
type ResourceOAuth2Model struct {
	Scopes types.List `tfsdk:"scopes"`
}

func (m ResourceOAuth2Model) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"scopes": types.ListType{ElemType: types.StringType},
	}
}

func (r *ResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *ResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Keycard resource. A resource is a system that exposes protected information or functionality requiring authentication.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this resource belongs to. Changing this will replace the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the resource.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the resource's purpose.",
				Optional:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the resource, typically its URL or URN. Must be unique within the zone.",
				Required:            true,
			},
			"credential_provider_id": schema.StringAttribute{
				MarkdownDescription: "The provider that issues credentials for accessing this resource.",
				Required:            true,
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The application that provides this resource.",
				Optional:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata associated with the resource.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"docs_url": schema.StringAttribute{
						MarkdownDescription: "URL to documentation relevant to this resource.",
						Optional:            true,
					},
				},
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth2 configuration for the resource.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"scopes": schema.ListAttribute{
						MarkdownDescription: "OAuth2 scopes required to access this resource. Must match scopes configured in the authorization server.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *ResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.KeycardClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.KeycardClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the create request
	credProviderID := data.CredentialProviderID.ValueString()
	createReq := client.ResourceCreate{
		Name:                 data.Name.ValueString(),
		Identifier:           data.Identifier.ValueString(),
		CredentialProviderId: &credProviderID,
	}

	// Set description if provided
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		createReq.Description = StringValueNullable(data.Description)
	}

	// Set application_id if provided
	if !data.ApplicationID.IsNull() && !data.ApplicationID.IsUnknown() {
		createReq.ApplicationId = data.ApplicationID.ValueStringPointer()
	}

	// Set metadata if provided
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		var metadataData ResourceMetadataModel
		diags := data.Metadata.As(ctx, &metadataData, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !metadataData.DocsURL.IsNull() && !metadataData.DocsURL.IsUnknown() {
			docsURL := metadataData.DocsURL.ValueString()
			createReq.Metadata = &client.Metadata{
				DocsUrl: &docsURL,
			}
		}
	}

	// Set scopes if provided (from oauth2 block)
	if !data.OAuth2.IsNull() && !data.OAuth2.IsUnknown() {
		var oauth2Data ResourceOAuth2Model
		diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !oauth2Data.Scopes.IsNull() && !oauth2Data.Scopes.IsUnknown() {
			var scopes []string
			diags := oauth2Data.Scopes.ElementsAs(ctx, &scopes, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			createReq.Scopes = &scopes
		}
	}

	// Create the resource
	createResp, err := r.client.CreateResourceWithResponse(ctx, data.ZoneID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create resource, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create resource, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create resource, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateResourceModelFromAPIResponse(ctx, createResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the resource
	getResp, err := r.client.GetResourceWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read resource, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Resource was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read resource, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read resource, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateResourceModelFromAPIResponse(ctx, getResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the update request
	updateReq := client.ResourceUpdate{}

	// Set name if changed
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		updateReq.Name = &name
	}

	// Set description (including null to remove it)
	if !data.Description.IsUnknown() {
		updateReq.Description = StringValueNullable(data.Description)
	}

	// Set identifier if changed
	if !data.Identifier.IsNull() && !data.Identifier.IsUnknown() {
		updateReq.Identifier = data.Identifier.ValueStringPointer()
	}

	// Set credential_provider_id (including null to remove it)
	if !data.CredentialProviderID.IsUnknown() {
		updateReq.CredentialProviderId = StringValueNullable(data.CredentialProviderID)
	}

	// Set application_id (including null to remove it)
	if !data.ApplicationID.IsUnknown() {
		updateReq.ApplicationId = StringValueNullable(data.ApplicationID)
	}

	// Set metadata if provided
	if !data.Metadata.IsUnknown() {
		if data.Metadata.IsNull() {
			// Remove metadata by setting to null
			updateReq.Metadata = nullable.NewNullNullable[client.MetadataUpdate]()
		} else {
			var metadataData ResourceMetadataModel
			diags := data.Metadata.As(ctx, &metadataData, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			metadataUpdate := client.MetadataUpdate{
				DocsUrl: StringValueNullable(metadataData.DocsURL),
			}
			updateReq.Metadata = nullable.NewNullableWithValue(metadataUpdate)
		}
	}

	// Set scopes if provided (from oauth2 block)
	if !data.OAuth2.IsUnknown() {
		if data.OAuth2.IsNull() {
			// Remove scopes
			updateReq.Scopes = nullable.NewNullNullable[[]string]()
		} else {
			var oauth2Data ResourceOAuth2Model
			diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if !oauth2Data.Scopes.IsUnknown() {
				if oauth2Data.Scopes.IsNull() {
					// Remove scopes
					updateReq.Scopes = nullable.NewNullNullable[[]string]()
				} else {
					var scopes []string
					diags := oauth2Data.Scopes.ElementsAs(ctx, &scopes, false)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					updateReq.Scopes = nullable.NewNullableWithValue(scopes)
				}
			}
		}
	}

	// Update the resource
	updateResp, err := r.client.UpdateResourceWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update resource, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update resource, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update resource, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateResourceModelFromAPIResponse(ctx, updateResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the resource
	deleteResp, err := r.client.DeleteResourceWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete resource, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete resource, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *ResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/resources/{resource-id}
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 || parts[0] != "zones" || parts[2] != "resources" || parts[1] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zones/{zone-id}/resources/{resource-id}', got: %s", req.ID),
		)
		return
	}

	zoneID := parts[1]
	resourceID := parts[3]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), resourceID)...)
}

// updateResourceModelFromAPIResponse updates a ResourceModel from an API Resource response.
func updateResourceModelFromAPIResponse(ctx context.Context, apiResource *client.Resource, data *ResourceModel) (diags diag.Diagnostics) {
	// Set basic fields
	data.ID = types.StringValue(apiResource.Id)
	data.ZoneID = types.StringValue(apiResource.ZoneId)
	data.Name = types.StringValue(apiResource.Name)
	data.Identifier = types.StringValue(apiResource.Identifier)
	data.Description = NullableStringValue(apiResource.Description)
	data.CredentialProviderID = types.StringPointerValue(apiResource.CredentialProviderId)
	data.ApplicationID = types.StringPointerValue(apiResource.ApplicationId)

	// Set metadata
	if apiResource.Metadata != nil && apiResource.Metadata.DocsUrl != nil {
		metadataModel := ResourceMetadataModel{
			DocsURL: types.StringPointerValue(apiResource.Metadata.DocsUrl),
		}
		metadataObj, d := types.ObjectValueFrom(ctx, metadataModel.AttributeTypes(), metadataModel)
		diags = append(diags, d...)
		data.Metadata = metadataObj
	} else {
		data.Metadata = types.ObjectNull(ResourceMetadataModel{}.AttributeTypes())
	}

	// Set scopes in oauth2 block
	scopes, err := apiResource.Scopes.Get()
	if err == nil {
		if len(scopes) > 0 {
			scopesList, d := types.ListValueFrom(ctx, types.StringType, scopes)
			diags.Append(d...)

			oauth2Model := ResourceOAuth2Model{
				Scopes: scopesList,
			}
			oauth2Obj, d := types.ObjectValueFrom(ctx, oauth2Model.AttributeTypes(), oauth2Model)
			diags.Append(d...)
			data.OAuth2 = oauth2Obj
		} else {
			data.OAuth2 = types.ObjectNull(ResourceOAuth2Model{}.AttributeTypes())
		}
	} else {
		data.OAuth2 = types.ObjectNull(ResourceOAuth2Model{}.AttributeTypes())
	}

	return
}
