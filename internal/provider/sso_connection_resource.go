package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &SSOConnectionResource{}
	_ resource.ResourceWithImportState = &SSOConnectionResource{}
)

func NewSSOConnectionResource() resource.Resource {
	return &SSOConnectionResource{}
}

// SSOConnectionResource defines the resource implementation.
type SSOConnectionResource struct {
	client *client.ClientWithResponses
}

// SSOConnectionResourceModel describes the resource data model.
type SSOConnectionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Identifier     types.String `tfsdk:"identifier"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
}

func (r *SSOConnectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_connection"
}

func (r *SSOConnectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an SSO connection for a Keycard organization. An SSO connection enables single sign-on authentication for organization members using an external identity provider (e.g., Okta, Azure AD).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the SSO connection.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The organization this SSO connection belongs to. Can be the organization ID or label. Changing this will replace the SSO connection.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "SSO provider identifier (e.g., the issuer URL from your identity provider).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(2048),
				},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth 2.0 client ID from your identity provider.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth 2.0 client secret from your identity provider.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *SSOConnectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SSOConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SSOConnectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.SSOConnectionCreate{
		Identifier: data.Identifier.ValueString(),
		ClientId:   data.ClientID.ValueString(),
	}

	if !data.ClientSecret.IsNull() && !data.ClientSecret.IsUnknown() {
		clientSecret := data.ClientSecret.ValueString()
		createReq.ClientSecret = &clientSecret
	}

	createResp, err := r.client.EnableSSOConnectionWithResponse(ctx, data.OrganizationID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create SSO connection, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create SSO connection, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create SSO connection, no response body")
		return
	}

	ssoConn := createResp.JSON201
	data.ID = types.StringValue(ssoConn.Id)
	data.Identifier = types.StringValue(ssoConn.Identifier)
	if ssoConn.ClientId.IsSpecified() && !ssoConn.ClientId.IsNull() {
		data.ClientID = types.StringValue(ssoConn.ClientId.MustGet())
	}
	// client_secret is write-only, preserve the configured value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SSOConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SSOConnectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.client.GetSSOConnectionWithResponse(ctx, data.OrganizationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SSO connection, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read SSO connection, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read SSO connection, no response body")
		return
	}

	ssoConn := getResp.JSON200
	data.ID = types.StringValue(ssoConn.Id)
	data.Identifier = types.StringValue(ssoConn.Identifier)
	if ssoConn.ClientId.IsSpecified() && !ssoConn.ClientId.IsNull() {
		data.ClientID = types.StringValue(ssoConn.ClientId.MustGet())
	}
	// client_secret is write-only, preserve state value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SSOConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SSOConnectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.SSOConnectionUpdate{}

	if !data.Identifier.IsNull() && !data.Identifier.IsUnknown() {
		identifier := data.Identifier.ValueString()
		updateReq.Identifier = &identifier
	}

	if !data.ClientID.IsNull() && !data.ClientID.IsUnknown() {
		clientID := data.ClientID.ValueString()
		updateReq.ClientId = &clientID
	}

	if !data.ClientSecret.IsNull() && !data.ClientSecret.IsUnknown() {
		clientSecret := data.ClientSecret.ValueString()
		updateReq.ClientSecret = &clientSecret
	}

	updateResp, err := r.client.UpdateSSOConnectionWithResponse(ctx, data.OrganizationID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update SSO connection, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update SSO connection, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update SSO connection, no response body")
		return
	}

	ssoConn := updateResp.JSON200
	data.ID = types.StringValue(ssoConn.Id)
	data.Identifier = types.StringValue(ssoConn.Identifier)
	if ssoConn.ClientId.IsSpecified() && !ssoConn.ClientId.IsNull() {
		data.ClientID = types.StringValue(ssoConn.ClientId.MustGet())
	}
	// client_secret is write-only, preserve the configured value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SSOConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SSOConnectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.client.DisableSSOConnectionWithResponse(ctx, data.OrganizationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete SSO connection, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete SSO connection, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *SSOConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is the organization ID (or label)
	organizationID := req.ID
	if organizationID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be the organization ID or label",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationID)...)

	// Fetch the SSO connection to get the ID
	getResp, err := r.client.GetSSOConnectionWithResponse(ctx, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SSO connection during import, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("SSO connection not found for organization: %s", organizationID))
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read SSO connection during import, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read SSO connection during import, no response body")
		return
	}

	ssoConn := getResp.JSON200
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ssoConn.Id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identifier"), ssoConn.Identifier)...)
	if ssoConn.ClientId.IsSpecified() && !ssoConn.ClientId.IsNull() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_id"), ssoConn.ClientId.MustGet())...)
	}
}
