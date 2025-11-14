package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = &ApplicationResource{}
	_ resource.ResourceWithImportState = &ApplicationResource{}
)

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

// ApplicationResource defines the resource implementation.
type ApplicationResource struct {
	client *client.KeycardClient
}

// ApplicationModel describes the application data model.
// This model is shared between the resource and data source.
type ApplicationModel struct {
	ID          types.String `tfsdk:"id"`
	ZoneID      types.String `tfsdk:"zone_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Identifier  types.String `tfsdk:"identifier"`
	Metadata    types.Object `tfsdk:"metadata"`
	OAuth2      types.Object `tfsdk:"oauth2"`
}

// ApplicationMetadataModel describes the nested metadata block data model.
type ApplicationMetadataModel struct {
	DocsURL types.String `tfsdk:"docs_url"`
}

func (m ApplicationMetadataModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"docs_url": types.StringType,
	}
}

// ApplicationOAuth2Model describes the nested oauth2 block data model.
type ApplicationOAuth2Model struct {
	RedirectURIs types.List `tfsdk:"redirect_uris"`
}

func (m ApplicationOAuth2Model) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"redirect_uris": types.ListType{ElemType: types.StringType},
	}
}

func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Keycard application. An application is a software system with an associated identity that can access resources.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the application.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this application belongs to. Changing this will replace the application.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the application.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the application's purpose.",
				Optional:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the application, typically its URL or URN. Must be unique within the zone.",
				Required:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata associated with the application.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"docs_url": schema.StringAttribute{
						MarkdownDescription: "URL to documentation relevant to this application.",
						Optional:            true,
					},
				},
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth2 configuration for the application.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_uris": schema.ListAttribute{
						MarkdownDescription: "OAuth 2.0 redirect URIs for authorization code/token delivery. Required if the application will perform user login flows.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *ApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the create request
	createReq := client.ApplicationCreate{
		Name:       data.Name.ValueString(),
		Identifier: data.Identifier.ValueString(),
	}

	// Set description if provided
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		createReq.Description = nullable.NewNullableWithValue(desc)
	}

	// Set metadata if provided
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		var metadataData ApplicationMetadataModel
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

	// Set OAuth2 configuration if provided
	if !data.OAuth2.IsNull() && !data.OAuth2.IsUnknown() {
		var oauth2Data ApplicationOAuth2Model
		diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !oauth2Data.RedirectURIs.IsNull() && !oauth2Data.RedirectURIs.IsUnknown() {
			var redirectURIs []string
			diags := oauth2Data.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			createReq.Protocols = &client.ApplicationProtocolCreate{
				Oauth2: &client.ApplicationOAuth2ProtocolCreate{
					RedirectUris: &redirectURIs,
				},
			}
		}
	}

	// Create the application
	createResp, err := r.client.CreateApplicationWithResponse(ctx, data.ZoneID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create application, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create application, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create application, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationModelFromAPIResponse(ctx, createResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the application
	getResp, err := r.client.GetApplicationWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Application was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read application, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read application, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationModelFromAPIResponse(ctx, getResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ApplicationModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the update request
	updateReq := client.ApplicationUpdate{}

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
		identifier := data.Identifier.ValueString()
		updateReq.Identifier = &identifier
	}

	// Set metadata if provided
	if !data.Metadata.IsUnknown() {
		if data.Metadata.IsNull() {
			// Remove metadata by setting to null
			updateReq.Metadata = nullable.NewNullNullable[client.MetadataUpdate]()
		} else {
			var metadataData ApplicationMetadataModel
			diags := data.Metadata.As(ctx, &metadataData, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			metadataUpdate := client.MetadataUpdate{}
			if !metadataData.DocsURL.IsUnknown() {
				metadataUpdate.DocsUrl = StringValueNullable(metadataData.DocsURL)
			}
			updateReq.Metadata = nullable.NewNullableWithValue(metadataUpdate)
		}
	}

	// Set OAuth2 configuration if provided
	if !data.OAuth2.IsUnknown() {
		if data.OAuth2.IsNull() {
			// Remove OAuth2 config
			updateReq.Protocols = nullable.NewNullNullable[client.ApplicationProtocolUpdate]()
		} else {
			var oauth2Data ApplicationOAuth2Model
			diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			protocolUpdate := client.ApplicationProtocolUpdate{
				Oauth2: nullable.Nullable[client.ApplicationOAuth2ProtocolUpdate]{},
			}

			if !oauth2Data.RedirectURIs.IsUnknown() {
				if oauth2Data.RedirectURIs.IsNull() {
					// Remove redirect URIs
					oauth2Update := client.ApplicationOAuth2ProtocolUpdate{
						RedirectUris: nullable.NewNullNullable[[]string](),
					}
					protocolUpdate.Oauth2 = nullable.NewNullableWithValue(oauth2Update)
				} else {
					var redirectURIs []string
					diags := oauth2Data.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					oauth2Update := client.ApplicationOAuth2ProtocolUpdate{
						RedirectUris: nullable.NewNullableWithValue(redirectURIs),
					}
					protocolUpdate.Oauth2 = nullable.NewNullableWithValue(oauth2Update)
				}
			}

			updateReq.Protocols = nullable.NewNullableWithValue(protocolUpdate)
		}
	}

	// Update the application
	updateResp, err := r.client.UpdateApplicationWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update application, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update application, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update application, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationModelFromAPIResponse(ctx, updateResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the application
	deleteResp, err := r.client.DeleteApplicationWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete application, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *ApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/applications/{application-id}
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 || parts[0] != "zones" || parts[2] != "applications" || parts[1] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zones/{zone-id}/applications/{application-id}', got: %s", req.ID),
		)
		return
	}

	zoneID := parts[1]
	applicationID := parts[3]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), applicationID)...)
}
