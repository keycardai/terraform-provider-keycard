package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	_ resource.Resource                = &PolicyResource{}
	_ resource.ResourceWithImportState = &PolicyResource{}
)

// NewPolicyResource builds the resource with the given Read retry window. Zero
// selects the full apiRetryWindow: a false 404 on a resource Read removes the
// resource from state and forces a destructive recreate, so production must
// wait out replica lag. Tests exercising a genuine miss inject a short window.
func NewPolicyResource(retryWindow time.Duration) resource.Resource {
	if retryWindow <= 0 {
		retryWindow = apiRetryWindow
	}
	return &PolicyResource{retryWindow: retryWindow}
}

// PolicyResource defines the resource implementation.
type PolicyResource struct {
	client      *client.ClientWithResponses
	retryWindow time.Duration
}

// PolicyModel describes the policy data model, shared by the resource and data source.
type PolicyModel struct {
	ID              types.String `tfsdk:"id"`
	ZoneID          types.String `tfsdk:"zone_id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	OwnerType       types.String `tfsdk:"owner_type"`
	CreatedBy       types.String `tfsdk:"created_by"`
	LatestVersionID types.String `tfsdk:"latest_version_id"`
}

func (r *PolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Keycard policy. A policy is a container for Cedar authorization rules within a zone; its Cedar content is managed separately through immutable policy versions.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this policy belongs to. Changing this will replace the policy.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the policy. Must be unique within the zone.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the policy's purpose.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"owner_type": schema.StringAttribute{
				MarkdownDescription: "Who manages this policy: `platform` (managed by Keycard) or `customer` (managed by the tenant).",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Identifier of the actor that created the policy.",
				Computed:            true,
			},
			"latest_version_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the policy's latest version. Null when the policy has no published versions.",
				Computed:            true,
			},
		},
	}
}

func (r *PolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreatePolicyRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
	}

	// A zone is provisioned asynchronously across services, so the policies
	// endpoint may 404 ("zone not found") briefly after the zone is created.
	createResp, err := callWithRetry(ctx, func() (*client.CreatePolicyResponse, error) {
		return r.client.CreatePolicyWithResponse(ctx, data.ZoneID.ValueString(), createReq)
	}, retryOnNotFound)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create policy, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create policy, no response body")
		return
	}

	updatePolicyModelFromAPIResponse(createResp.JSON201, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// svc-pdp is eventually consistent: a policy just created (or its zone
	// still provisioning) can 404 on a read replica for a while. Retry
	// transient 404s over the configured window; acting on a false 404 here
	// removes the resource from state and forces a destructive recreate, so
	// production waits out replica lag rather than surfacing a genuine miss
	// quickly.
	getResp, err := callWithRetry(ctx, func() (*client.GetPolicyResponse, error) {
		return r.client.GetPolicyWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	}, retryOnNotFound, withRetryWindow(r.retryWindow))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Policy was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read policy, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read policy, no response body")
		return
	}

	// Archiving is a soft-delete: reads of an archived policy may return 200
	// with archived_at set rather than 404.
	if isArchived(getResp.JSON200.ArchivedAt) {
		resp.State.RemoveResource(ctx)
		return
	}

	updatePolicyModelFromAPIResponse(getResp.JSON200, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdatePolicyRequest{
		Name:        data.Name.ValueStringPointer(),
		Description: data.Description.ValueStringPointer(),
	}

	// svc-pdp is eventually consistent: a PATCH can land on a replica that
	// hasn't yet observed the create, returning a transient 404 for a policy
	// we know exists. Retry those; a persistent 404 surfaces below.
	updateResp, err := callWithRetry(ctx, func() (*client.UpdatePolicyResponse, error) {
		return r.client.UpdatePolicyWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString(), updateReq)
	}, retryOnNotFound)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update policy, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update policy, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update policy, no response body")
		return
	}

	updatePolicyModelFromAPIResponse(updateResp.JSON200, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.client.ArchivePolicyWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete policy, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 200 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete policy, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/policies/{policy-id}
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 || parts[0] != "zones" || parts[2] != "policies" || parts[1] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zones/{zone-id}/policies/{policy-id}', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}
