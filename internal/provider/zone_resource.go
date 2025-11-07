package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ZoneResource{}
	_ resource.ResourceWithImportState = &ZoneResource{}
)

func NewZoneResource() resource.Resource {
	return &ZoneResource{}
}

// ZoneResource defines the resource implementation.
type ZoneResource struct {
	client *client.ClientWithResponses
}

// ZoneResourceModel describes the resource data model.
type ZoneResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	OAuth2        types.Object `tfsdk:"oauth2"`
	EncryptionKey types.Object `tfsdk:"encryption_key"`
}

// OAuth2Model describes the nested oauth2 block data model.
type OAuth2Model struct {
	PkceRequired types.Bool   `tfsdk:"pkce_required"`
	DcrEnabled   types.Bool   `tfsdk:"dcr_enabled"`
	IssuerUri    types.String `tfsdk:"issuer_uri"`
	RedirectUri  types.String `tfsdk:"redirect_uri"`
}

func (m OAuth2Model) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"pkce_required": types.BoolType,
		"dcr_enabled":   types.BoolType,
		"issuer_uri":    types.StringType,
		"redirect_uri":  types.StringType,
	}
}

// EncryptionKeyConfigModel describes the encryption_key nested block data model.
type EncryptionKeyConfigModel struct {
	Aws types.Object `tfsdk:"aws"`
}

func (m EncryptionKeyConfigModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"aws": types.ObjectType{AttrTypes: EncryptionKeyAwsKmsConfigModel{}.AttributeTypes()},
	}
}

// EncryptionKeyAwsKmsConfigModel describes the encryption_key.aws nested block data model.
type EncryptionKeyAwsKmsConfigModel struct {
	Arn types.String `tfsdk:"arn"`
}

func (m EncryptionKeyAwsKmsConfigModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"arn": types.StringType,
	}
}

func (r *ZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (r *ZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Keycard zone. Zones are isolated environments for organizing IAM resources within an organization.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the zone.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the zone.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the zone's purpose.",
				Optional:            true,
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth2 configuration for the zone.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"pkce_required": schema.BoolAttribute{
						MarkdownDescription: "Whether PKCE (Proof Key for Code Exchange) is required for authorization code flows. Defaults to true.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"dcr_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether Dynamic Client Registration (DCR) is enabled. Defaults to true.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"issuer_uri": schema.StringAttribute{
						MarkdownDescription: "OAuth 2.0 issuer URI for this zone.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"redirect_uri": schema.StringAttribute{
						MarkdownDescription: "OAuth 2.0 redirect URI for this zone.",
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
			"encryption_key": schema.SingleNestedAttribute{
				MarkdownDescription: "Encryption key configuration for the zone. Changing this value will force replacement of the zone.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"aws": schema.SingleNestedAttribute{
						MarkdownDescription: "AWS KMS configuration for encryption.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"arn": schema.StringAttribute{
								MarkdownDescription: "ARN of the AWS KMS key to use for encryption.",
								Required:            true,
							},
						},
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// updateZoneModelFromAPIResponse maps a Zone API response to the ZoneResourceModel.
// This is a shared helper function used by Create, Read, and Update operations.
// It returns any diagnostics encountered during the mapping.
func updateZoneModelFromAPIResponse(ctx context.Context, zone *client.Zone, data *ZoneResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(zone.Id)
	data.Name = types.StringValue(zone.Name)
	data.Description = NullableStringValue(zone.Description)

	oauth2Data := OAuth2Model{
		PkceRequired: types.BoolValue(zone.Protocols.Oauth2.PkceRequired),
		DcrEnabled:   types.BoolValue(zone.Protocols.Oauth2.DcrEnabled),
		IssuerUri:    types.StringValue(zone.Protocols.Oauth2.Issuer),
		RedirectUri:  types.StringValue(zone.Protocols.Oauth2.RedirectUri),
	}

	oauth2Obj, objDiags := types.ObjectValueFrom(ctx, oauth2Data.AttributeTypes(), oauth2Data)
	diags.Append(objDiags...)
	data.OAuth2 = oauth2Obj

	// Handle encryption_key if present in API response
	if zone.EncryptionKey != nil {
		// Transform from API format {type: "aws", arn: "..."} to Terraform format {aws: {arn: "..."}}
		if zone.EncryptionKey.Type == "aws" {
			awsKmsData := EncryptionKeyAwsKmsConfigModel{
				Arn: types.StringValue(zone.EncryptionKey.Arn),
			}

			awsKmsObj, awsObjDiags := types.ObjectValueFrom(ctx, awsKmsData.AttributeTypes(), awsKmsData)
			diags.Append(awsObjDiags...)

			encryptionKeyData := EncryptionKeyConfigModel{
				Aws: awsKmsObj,
			}

			encryptionKeyObj, encObjDiags := types.ObjectValueFrom(ctx, encryptionKeyData.AttributeTypes(), encryptionKeyData)
			diags.Append(encObjDiags...)
			data.EncryptionKey = encryptionKeyObj
		}
	} else {
		// No encryption_key in API response
		data.EncryptionKey = types.ObjectNull(EncryptionKeyConfigModel{}.AttributeTypes())
	}

	return diags
}

func (r *ZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the create request
	createReq := client.ZoneCreate{
		Name: data.Name.ValueString(),
	}

	// Set description if provided
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		createReq.Description = nullable.NewNullableWithValue(desc)
	}

	// Set OAuth2 configuration if provided
	if !data.OAuth2.IsNull() && !data.OAuth2.IsUnknown() {
		var oauth2Data OAuth2Model
		diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		createReq.Protocols = &client.ZoneProtocolCreate{
			Oauth2: &client.ZoneOAuth2ProtocolCreate{},
		}

		if !oauth2Data.PkceRequired.IsNull() && !oauth2Data.PkceRequired.IsUnknown() {
			createReq.Protocols.Oauth2.PkceRequired = oauth2Data.PkceRequired.ValueBoolPointer()
		}

		if !oauth2Data.DcrEnabled.IsNull() && !oauth2Data.DcrEnabled.IsUnknown() {
			createReq.Protocols.Oauth2.DcrEnabled = oauth2Data.DcrEnabled.ValueBoolPointer()
		}
	}

	// Set encryption_key configuration if provided
	if !data.EncryptionKey.IsNull() && !data.EncryptionKey.IsUnknown() {
		var encryptionKeyData EncryptionKeyConfigModel
		diags := data.EncryptionKey.As(ctx, &encryptionKeyData, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Extract the AWS KMS configuration
		if !encryptionKeyData.Aws.IsNull() && !encryptionKeyData.Aws.IsUnknown() {
			var awsData EncryptionKeyAwsKmsConfigModel
			diags := encryptionKeyData.Aws.As(ctx, &awsData, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// Transform to API format: {type: "aws", arn: "..."}
			createReq.EncryptionKey = &client.EncryptionKeyAwsKmsConfig{
				Type: client.Aws,
				Arn:  awsData.Arn.ValueString(),
			}
		}
	}

	// Create the zone
	createResp, err := r.client.CreateZoneWithResponse(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create zone, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create zone, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create zone, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateZoneModelFromAPIResponse(ctx, createResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the zone
	getResp, err := r.client.GetZoneWithResponse(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read zone, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Zone was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read zone, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read zone, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateZoneModelFromAPIResponse(ctx, getResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ZoneResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the update request
	updateReq := client.ZoneUpdate{}

	// Set name if changed
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		updateReq.Name = &name
	}

	// Set description (including null to remove it)
	if !data.Description.IsUnknown() {
		updateReq.Description = StringValueNullable(data.Description)
	}

	// Set OAuth2 configuration if provided
	if !data.OAuth2.IsUnknown() {
		var oauth2Data OAuth2Model
		diags := data.OAuth2.As(ctx, &oauth2Data, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		protocolUpdate := client.ZoneProtocolUpdate{
			Oauth2: &client.ZoneOAuth2ProtocolUpdate{},
		}

		if !oauth2Data.PkceRequired.IsNull() && !oauth2Data.PkceRequired.IsUnknown() {
			protocolUpdate.Oauth2.PkceRequired = BoolValueNullable(oauth2Data.PkceRequired)
		}

		if !oauth2Data.DcrEnabled.IsNull() && !oauth2Data.DcrEnabled.IsUnknown() {
			protocolUpdate.Oauth2.DcrEnabled = BoolValueNullable(oauth2Data.DcrEnabled)
		}

		updateReq.Protocols = nullable.NewNullableWithValue(protocolUpdate)
	}

	// Update the zone
	updateResp, err := r.client.UpdateZoneWithResponse(ctx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update zone, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update zone, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update zone, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateZoneModelFromAPIResponse(ctx, updateResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the zone
	deleteResp, err := r.client.DeleteZoneWithResponse(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete zone, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete zone, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *ZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
