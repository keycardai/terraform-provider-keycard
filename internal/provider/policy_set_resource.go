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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &PolicySetResource{}
	_ resource.ResourceWithImportState = &PolicySetResource{}
)

// NewPolicySetResource builds the resource with the given Read retry window. Zero
// selects the full apiRetryWindow: a false 404 on a resource Read removes the
// resource from state and forces a destructive recreate, so production must
// wait out replica lag. Tests exercising a genuine miss inject a short window.
func NewPolicySetResource(retryWindow time.Duration) resource.Resource {
	if retryWindow <= 0 {
		retryWindow = apiRetryWindow
	}
	return &PolicySetResource{retryWindow: retryWindow}
}

// PolicySetResource defines the resource implementation.
type PolicySetResource struct {
	client      *client.ClientWithResponses
	retryWindow time.Duration
}

// PolicySetModel describes the policy set data model, shared by the resource and data source.
type PolicySetModel struct {
	ID              types.String `tfsdk:"id"`
	ZoneID          types.String `tfsdk:"zone_id"`
	Name            types.String `tfsdk:"name"`
	TargetType      types.String `tfsdk:"target_type"`
	ETag            types.String `tfsdk:"etag"`
	OwnerType       types.String `tfsdk:"owner_type"`
	CreatedBy       types.String `tfsdk:"created_by"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	LatestVersionID types.String `tfsdk:"latest_version_id"`
	LatestVersion   types.Int64  `tfsdk:"latest_version"`
	Active          types.Bool   `tfsdk:"active"`
	ActiveVersionID types.String `tfsdk:"active_version_id"`
	ActiveVersion   types.Int64  `tfsdk:"active_version"`
	ShadowVersionID types.String `tfsdk:"shadow_version_id"`
	ShadowVersion   types.Int64  `tfsdk:"shadow_version"`
	TargetID        types.String `tfsdk:"target_id"`
	Mode            types.String `tfsdk:"mode"`
}

func (r *PolicySetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_set"
}

func (r *PolicySetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Keycard policy set. A policy set is a container that binds policy set versions to a zone or user scope; its versions are managed separately.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy set.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this policy set belongs to. Changing this will replace the policy set.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the policy set. Must be unique within the zone.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"target_type": schema.StringAttribute{
				MarkdownDescription: "What this policy set targets: `zone` (applies to all requests in the zone) or `user` (scoped to a specific user). Defaults to `zone`. Changing this will replace the policy set.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("zone"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("zone", "user"),
				},
			},
			"etag": schema.StringAttribute{
				MarkdownDescription: "Entity tag for optimistic concurrency control, sent as `If-Match` on updates.",
				Computed:            true,
			},
			"owner_type": schema.StringAttribute{
				MarkdownDescription: "Who manages this policy set: `platform` (managed by Keycard) or `customer` (managed by the tenant).",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Identifier of the actor that created the policy set.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the policy set was created (RFC 3339).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the policy set was last updated (RFC 3339).",
				Computed:            true,
			},
			"latest_version_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the policy set's latest version. Null when the policy set has no versions.",
				Computed:            true,
			},
			"latest_version": schema.Int64Attribute{
				MarkdownDescription: "Human-readable version number of the latest version. Null when the policy set has no versions.",
				Computed:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether this policy set is currently bound to a scope.",
				Computed:            true,
			},
			"active_version_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the currently active (bound) version. Null when unbound.",
				Computed:            true,
			},
			"active_version": schema.Int64Attribute{
				MarkdownDescription: "Human-readable version number of the active version. Null when unbound.",
				Computed:            true,
			},
			"shadow_version_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the shadow (observed) version, if any.",
				Computed:            true,
			},
			"shadow_version": schema.Int64Attribute{
				MarkdownDescription: "Human-readable version number of the shadow version, if any.",
				Computed:            true,
			},
			"target_id": schema.StringAttribute{
				MarkdownDescription: "Target entity ID of the active binding. Equals `zone_id` for zone-targeted sets; the principal identifier for user-targeted sets. Null when unbound.",
				Computed:            true,
			},
			"mode": schema.StringAttribute{
				MarkdownDescription: "Binding mode of the active binding: `active` or `shadow`. Null when unbound.",
				Computed:            true,
			},
		},
	}
}

func (r *PolicySetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicySetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicySetModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetType := client.CreatePolicySetRequestTargetType(data.TargetType.ValueString())
	createReq := client.CreatePolicySetRequest{
		Name:       data.Name.ValueString(),
		TargetType: &targetType,
	}

	// A zone is provisioned asynchronously across services, so the policy-sets
	// endpoint may 404 ("zone not found") briefly after the zone is created.
	createResp, err := callWithRetry(ctx, func() (*client.CreatePolicySetResponse, error) {
		return r.client.CreatePolicySetWithResponse(ctx, data.ZoneID.ValueString(), createReq)
	}, retryOnNotFound)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy set, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create policy set, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create policy set, no response body")
		return
	}

	updatePolicySetModelFromAPIResponse(createResp.JSON201, &data)
	data.ETag = etagValue(createResp.HTTPResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicySetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicySetModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// svc-pdp is eventually consistent: a policy set just created (or its zone
	// still provisioning) can 404 on a read replica for a while. Retry
	// transient 404s over the configured window; acting on a false 404 here
	// removes the resource from state and forces a destructive recreate, so
	// production waits out replica lag rather than surfacing a genuine miss
	// quickly.
	getResp, err := callWithRetry(ctx, func() (*client.GetPolicySetResponse, error) {
		return r.client.GetPolicySetWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	}, retryOnNotFound, withRetryWindow(r.retryWindow))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy set, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Policy set was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read policy set, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read policy set, no response body")
		return
	}

	updatePolicySetModelFromAPIResponse(getResp.JSON200, &data)
	data.ETag = etagValue(getResp.HTTPResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicySetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PolicySetModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdatePolicySetRequest{
		Name: plan.Name.ValueStringPointer(),
	}

	zoneID := plan.ZoneID.ValueString()
	id := plan.ID.ValueString()

	// Optimistic concurrency: send the ETag captured in state as If-Match. The
	// ETag can be stale — svc-pdp is eventually consistent, so the refresh that
	// populated state may have read a replica that lagged a prior write, and
	// back-to-back applies compound that. A stale If-Match yields a 412; rather
	// than fail a legitimate update, refetch the current ETag and retry so the
	// write self-heals. svc-pdp also returns a transient 404 when a PATCH lands
	// on a replica that hasn't observed the create yet, so retry that too.
	ifMatch := state.ETag.ValueString()
	var etagRefreshErr error
	updateResp, err := callWithRetry(ctx, func() (*client.UpdatePolicySetResponse, error) {
		params := client.UpdatePolicySetParams{}
		if ifMatch != "" {
			params.IfMatch = &ifMatch
		}
		resp, err := r.client.UpdatePolicySetWithResponse(ctx, zoneID, id, &params, updateReq)
		if err == nil && resp.StatusCode() == 412 {
			// A refresh failure is not fatal: it may be as transient as the 412
			// itself, and the next iteration re-attempts it. Record it so a
			// retry loop stuck on a stale ETag can report the real cause.
			etag, rerr := r.currentPolicySetETag(ctx, zoneID, id)
			etagRefreshErr = rerr
			if rerr != nil {
				tflog.Warn(ctx, "Failed to refresh policy set ETag after 412; retrying with previous ETag", map[string]any{
					"policy_set_id": id,
					"error":         rerr.Error(),
				})
			} else {
				ifMatch = etag
			}
		}
		return resp, err
	}, retryOnNotFoundOrConflict)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update policy set, got error: %s", err))
		return
	}

	if updateResp.StatusCode() == 412 {
		detail := "The policy set was modified outside of Terraform since it was last read. Refresh state (terraform refresh or terraform apply -refresh-only) and retry."
		if etagRefreshErr != nil {
			detail = fmt.Sprintf("%s Refreshing the ETag during retries also failed: %s.", detail, etagRefreshErr)
		}
		resp.Diagnostics.AddError("Conflict", detail)
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update policy set, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update policy set, no response body")
		return
	}

	updatePolicySetModelFromAPIResponse(updateResp.JSON200, &plan)
	plan.ETag = etagValue(updateResp.HTTPResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// currentPolicySetETag fetches the policy set's current ETag, for retrying an
// If-Match update whose state ETag went stale.
func (r *PolicySetResource) currentPolicySetETag(ctx context.Context, zoneID, id string) (string, error) {
	getResp, err := r.client.GetPolicySetWithResponse(ctx, zoneID, id)
	if err != nil {
		return "", err
	}
	if getResp.StatusCode() != 200 {
		return "", fmt.Errorf("got status %d", getResp.StatusCode())
	}
	etag := getResp.HTTPResponse.Header.Get("ETag")
	if etag == "" {
		return "", fmt.Errorf("response has no ETag header")
	}
	return etag, nil
}

func (r *PolicySetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicySetModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.client.ArchivePolicySetWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete policy set, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 200 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete policy set, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *PolicySetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/policy-sets/{policy-set-id}
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 || parts[0] != "zones" || parts[2] != "policy-sets" || parts[1] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zones/{zone-id}/policy-sets/{policy-set-id}', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}
