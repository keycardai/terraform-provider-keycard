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
	_ resource.Resource                = &PolicySetActivationResource{}
	_ resource.ResourceWithImportState = &PolicySetActivationResource{}
)

// NewPolicySetActivationResource builds the resource with the given Read retry
// window. Zero selects the full apiRetryWindow: a false 404 on a resource Read
// removes the resource from state and forces a destructive recreate, so
// production must wait out replica lag. Tests exercising a genuine miss inject a
// short window.
func NewPolicySetActivationResource(readRetryWindow time.Duration) resource.Resource {
	if readRetryWindow <= 0 {
		readRetryWindow = apiRetryWindow
	}
	return &PolicySetActivationResource{readRetryWindow: readRetryWindow}
}

// PolicySetActivationResource defines the resource implementation.
type PolicySetActivationResource struct {
	client *client.ClientWithResponses
	// readRetryWindow bounds transient-404 retries in Read only. Create always
	// uses the full apiRetryWindow (async zone provisioning) and Update the short
	// notFoundRetryWindow (a persistent 404 is a genuine miss); neither is
	// test-tunable, by design.
	readRetryWindow time.Duration
}

// PolicySetActivationResourceModel describes the policy set activation data model.
type PolicySetActivationResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ZoneID             types.String `tfsdk:"zone_id"`
	PolicySetID        types.String `tfsdk:"policy_set_id"`
	PolicySetVersionID types.String `tfsdk:"policy_set_version_id"`
}

func (r *PolicySetActivationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_set_activation"
}

func (r *PolicySetActivationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Binds a `keycard_policy_set_version` as the active policy set for a zone. " +
			"A zone has at most one active binding, keyed by the zone itself: activating a version of a different policy set takes over the slot, so declare at most one `keycard_policy_set_activation` per zone. " +
			"Changing `policy_set_version_id` re-activates in place — roll forward or roll back — and the server atomically demotes the previously active version." +
			"\n\n~> **Destroying this resource does not unbind anything.** By design a zone always has an active policy set binding: bindings are replaced, never emptied (a zone without an active policy set fails closed), so no deactivation operation exists. Destroy removes the resource from Terraform state only; the version stays active server-side, and changing what is active means binding a different policy set version. " +
			"While a version is active, it, the policy versions it references, and its policy set cannot be archived — to destroy the whole stack, activate a replacement version outside the configuration first.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the activation; equals the zone ID, since a zone has a single active binding.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone whose active policy set binding this manages. Changing this creates a new activation.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"policy_set_id": schema.StringAttribute{
				MarkdownDescription: "The policy set the activated version belongs to. Changing it activates the referenced version in place; the new set takes over the zone's single active slot and the server atomically demotes the previously active version.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"policy_set_version_id": schema.StringAttribute{
				MarkdownDescription: "The policy set version to activate. Changing it activates the new version in place; the server atomically demotes the old one. Rolling back to an older version is supported. Cannot reference a shadow-bound version.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *PolicySetActivationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicySetActivationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicySetActivationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A zone is provisioned asynchronously across services, so the endpoint may
	// 404 ("zone not found") briefly after the zone is created. Retry over the
	// full window to cover that provisioning lag.
	if !r.activate(ctx, &data, resp.Diagnostics.AddError, apiRetryWindow) {
		return
	}

	data.ID = types.StringValue(data.ZoneID.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicySetActivationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicySetActivationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// svc-pdp is eventually consistent: a binding just written (or its zone still
	// provisioning) can lag on a read replica for a while. Retry transient 404s
	// over the configured window; acting on a false 404 here removes the resource
	// from state and forces a destructive recreate, so production waits out
	// replica lag rather than surfacing a genuine miss quickly.
	getResp, err := callWithRetry(ctx, func() (*client.GetPolicySetResponse, error) {
		return r.client.GetPolicySetWithResponse(ctx, data.ZoneID.ValueString(), data.PolicySetID.ValueString())
	}, retryOnNotFound, withRetryWindow(r.readRetryWindow))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy set activation, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Policy set was archived/deleted outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read policy set activation, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read policy set activation, no response body")
		return
	}

	if isArchived(getResp.JSON200.ArchivedAt) {
		resp.State.RemoveResource(ctx)
		return
	}

	// Gate drift on the required `active` flag, not on active_version_id: the
	// latter is nullable and non-required, so an unbound set may serialize it as
	// null or omit it entirely, and keying off its presence would miss drift.
	// `active` is always present.
	if !getResp.JSON200.Active {
		// No active binding for this set: deactivated out-of-band, or the zone's
		// single slot was taken over by a different policy set.
		resp.State.RemoveResource(ctx)
		return
	}

	if activeVersionID := getResp.JSON200.ActiveVersionId; activeVersionID.IsSpecified() && !activeVersionID.IsNull() {
		// Write the live active version into state. If it drifted from the
		// declared version (activated out-of-band), Terraform plans an in-place
		// Update to re-activate the declared version rather than a destructive
		// recreate. Also covers import, where only zone_id/policy_set_id are known.
		data.PolicySetVersionID = types.StringValue(activeVersionID.MustGet())
	} else {
		// active is true but active_version_id is absent/null (a server quirk):
		// keep the declared version rather than destructively re-activating. This
		// masks out-of-band version drift, so surface it.
		resp.Diagnostics.AddWarning(
			"API Response Incomplete",
			"Policy set is active but active_version_id was not returned; cannot verify the active version matches configuration.",
		)
	}

	data.ID = types.StringValue(data.ZoneID.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicySetActivationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PolicySetActivationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The common rollout is create_before_destroy on keycard_policy_set_version
	// followed by re-pointing this activation to the freshly created version in
	// the same apply. That PATCH can land on a replica that has not yet observed
	// the new version and return a transient 404, so retry — but over the short
	// window, since a persistent 404 here is a genuine miss (a version that does
	// not belong to this set) and should surface quickly rather than hang.
	// On error we return without setting state, so the framework keeps the
	// previously active version in state.
	if !r.activate(ctx, &data, resp.Diagnostics.AddError, notFoundRetryWindow) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete intentionally makes no API call. By design a zone's active policy set
// binding is replaced, never emptied — a zone without an active binding fails
// closed on evaluation, so no deactivation operation exists. Destroy only
// removes the binding from Terraform state; the version stays active
// server-side.
func (r *PolicySetActivationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *PolicySetActivationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zone_id/policy_set_id; Read resolves the currently
	// active version from the live binding.
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zone_id/policy_set_id', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_set_id"), parts[1])...)
}

// activate PATCHes the referenced version active and reports success. The
// server operation is idempotent and atomically demotes the previously active
// version. Transient 404s are retried over retryWindow: Create passes the full
// window (asynchronous zone provisioning), Update the short window (a freshly
// created version lagging a replica).
func (r *PolicySetActivationResource) activate(ctx context.Context, data *PolicySetActivationResourceModel, addError func(string, string), retryWindow time.Duration) bool {
	do := func() (*client.UpdatePolicySetVersionResponse, error) {
		return r.client.UpdatePolicySetVersionWithResponse(
			ctx,
			data.ZoneID.ValueString(),
			data.PolicySetID.ValueString(),
			data.PolicySetVersionID.ValueString(),
			client.UpdatePolicySetVersionRequest{Active: true},
		)
	}

	updateResp, err := callWithRetry(ctx, do, retryOnNotFound, withRetryWindow(retryWindow))
	if err != nil {
		addError("Client Error", fmt.Sprintf("Unable to activate policy set version, got error: %s", err))
		return false
	}

	switch {
	case updateResp.StatusCode() == 400:
		addError(
			"API Error",
			fmt.Sprintf("Unable to activate policy set version, got status 400: %s. A shadow-bound version cannot be activated; remove the shadow binding first.", string(updateResp.Body)),
		)
		return false
	case updateResp.StatusCode() == 404:
		addError(
			"API Error",
			fmt.Sprintf("Unable to activate policy set version, got status 404: %s. Verify that policy_set_version_id belongs to policy_set_id and that both exist in zone_id.", string(updateResp.Body)),
		)
		return false
	case updateResp.StatusCode() != 200:
		addError(
			"API Error",
			fmt.Sprintf("Unable to activate policy set version, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return false
	case updateResp.JSON200 == nil:
		addError("API Error", "Unable to activate policy set version, no response body")
		return false
	}

	return true
}
