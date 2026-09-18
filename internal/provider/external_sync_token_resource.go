package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

var _ resource.Resource = &ExternalSyncTokenResource{}

func NewExternalSyncTokenResource() resource.Resource {
	return &ExternalSyncTokenResource{}
}

type ExternalSyncTokenResource struct {
	client *client.ClientWithResponses
}

type ExternalSyncTokenModel struct {
	ID     types.String `tfsdk:"id"`
	ZoneID types.String `tfsdk:"zone_id"`
	Token  types.String `tfsdk:"token"`
}

func (r *ExternalSyncTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_sync_token"
}

func (r *ExternalSyncTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a bearer token an identity provider uses to provision users and groups into a zone over SCIM. " +
			"The zone must have `external_sync_enabled` set and a user identity provider configured. " +
			"**Important**: The token value is only available immediately after creation. " +
			"If Terraform state is lost, the resource must be recreated as the token cannot be retrieved. " +
			"~> **Note** Import is not supported for this resource.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the token.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this token provisions users into. Changing this will replace the token.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The bearer token to configure in the identity provider. Only returned on creation and cannot be retrieved later.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ExternalSyncTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ExternalSyncTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExternalSyncTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := r.client.CreateExternalSyncTokenWithResponse(ctx, data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create external sync token, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create external sync token, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create external sync token, no response body")
		return
	}

	data.ID = types.StringValue(createResp.JSON200.Id)
	data.ZoneID = types.StringValue(createResp.JSON200.ZoneId)
	data.Token = types.StringValue(createResp.JSON200.Token)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalSyncTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExternalSyncTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.client.GetExternalSyncTokenWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read external sync token, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read external sync token, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read external sync token, no response body")
		return
	}

	// token is only returned on create; keep the stored value.
	data.ID = types.StringValue(getResp.JSON200.Id)
	data.ZoneID = types.StringValue(getResp.JSON200.ZoneId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalSyncTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Unreachable: every settable attribute is RequiresReplace.
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"External sync tokens are immutable. Any changes require resource replacement.",
	)
}

func (r *ExternalSyncTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExternalSyncTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.client.DeleteExternalSyncTokenWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete external sync token, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete external sync token, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}
