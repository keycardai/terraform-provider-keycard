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
	_ resource.Resource                = &PolicyVersionResource{}
	_ resource.ResourceWithImportState = &PolicyVersionResource{}
)

// NewPolicyVersionResource builds the resource with the given Read retry window.
// Zero selects the full apiRetryWindow: a false 404 on a resource Read removes
// the resource from state and forces a destructive recreate, so production must
// wait out replica lag. Tests exercising a genuine miss inject a short window.
func NewPolicyVersionResource(retryWindow time.Duration) resource.Resource {
	if retryWindow <= 0 {
		retryWindow = apiRetryWindow
	}
	return &PolicyVersionResource{retryWindow: retryWindow}
}

// PolicyVersionResource defines the resource implementation.
type PolicyVersionResource struct {
	client      *client.ClientWithResponses
	retryWindow time.Duration
}

// PolicyVersionResourceModel describes the policy version data model.
type PolicyVersionResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ZoneID        types.String `tfsdk:"zone_id"`
	PolicyID      types.String `tfsdk:"policy_id"`
	Cedar         types.String `tfsdk:"cedar"`
	SchemaVersion types.String `tfsdk:"schema_version"`
	Version       types.Int64  `tfsdk:"version"`
	Sha           types.String `tfsdk:"sha"`
	CreatedAt     types.String `tfsdk:"created_at"`
	OwnerType     types.String `tfsdk:"owner_type"`
}

func (r *PolicyVersionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_version"
}

func (r *PolicyVersionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Publishes an immutable Cedar policy version under a `keycard_policy`. Any change to the content or schema creates a new version; changing an attribute replaces the resource. Use `lifecycle { create_before_destroy = true }` so the replacement version is published before the old one is archived.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy version.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this policy version belongs to. Changing this creates a new version.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_id": schema.StringAttribute{
				MarkdownDescription: "The policy this version is published under. Changing this creates a new version.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cedar": schema.StringAttribute{
				MarkdownDescription: "The Cedar policy content in human-readable syntax. Immutable: changing it publishes a new version. The server may normalize formatting; Terraform preserves the configured value to avoid spurious diffs.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"schema_version": schema.StringAttribute{
				MarkdownDescription: "The Cedar schema version this policy is validated against. Must not reference an archived schema. Changing it publishes a new version.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "Human-readable version number, incremented per published version (e.g., 1, 2, 3).",
				Computed:            true,
			},
			"sha": schema.StringAttribute{
				MarkdownDescription: "Hex-encoded hash of the stored (normalized) policy content.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the version was created (RFC 3339).",
				Computed:            true,
			},
			"owner_type": schema.StringAttribute{
				MarkdownDescription: "Who manages this policy version: `platform` (managed by Keycard) or `customer` (managed by the tenant).",
				Computed:            true,
			},
		},
	}
}

func (r *PolicyVersionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicyVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyVersionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()
	policyID := data.PolicyID.ValueString()
	schemaVersion := data.SchemaVersion.ValueString()

	// The server silently accepts deprecated schemas; warn at plan/apply time so
	// pinning a superseded schema is visible. Best-effort: a lookup failure is not
	// fatal to the create.
	if warning := r.deprecatedSchemaWarning(ctx, zoneID, schemaVersion); warning != "" {
		resp.Diagnostics.AddWarning("Deprecated Policy Schema", warning)
	}

	// cedar maps to the API's cedar_raw (human-readable form; cedar_json is
	// derived server-side). The request field is nullable and the server skips
	// content validation entirely when it's empty, so the schema's Required +
	// LengthAtLeast(1) are the only guards against publishing an empty version.
	createReq := client.CreatePolicyVersionRequest{
		CedarRaw:      stringValueNullable(data.Cedar),
		SchemaVersion: schemaVersion,
	}

	// A zone is provisioned asynchronously across services, so the policy versions
	// endpoint may 404 ("zone not found") briefly after the zone is created.
	createResp, err := callWithRetry(ctx, func() (*client.CreatePolicyVersionResponse, error) {
		return r.client.CreatePolicyVersionWithResponse(ctx, zoneID, policyID, createReq)
	}, retryOnNotFound)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy version, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create policy version, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create policy version, no response body")
		return
	}

	// Keep the configured cedar value; the response carries the server-normalized
	// form, which would otherwise produce a perpetual diff.
	updatePolicyVersionComputedFromAPIResponse(createResp.JSON201, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyVersionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// svc-pdp is eventually consistent: a version just created (or its zone still
	// provisioning) can 404 on a read replica for a while. Retry transient 404s
	// over the configured window; acting on a false 404 here removes the resource
	// from state and forces a destructive recreate, so production waits out
	// replica lag rather than surfacing a genuine miss quickly.
	//
	// API contract (svc-pdp GetPolicyVersion): with no format query param the
	// response populates both cedar_json and cedar_raw; cedar_raw is null only
	// when a caller explicitly narrows with format=json, which this client
	// never sends (the vendored spec doesn't expose the param). Import relies
	// on cedar_raw being present here.
	getResp, err := callWithRetry(ctx, func() (*client.GetPolicyVersionResponse, error) {
		return r.client.GetPolicyVersionWithResponse(ctx, data.ZoneID.ValueString(), data.PolicyID.ValueString(), data.ID.ValueString())
	}, retryOnNotFound, withRetryWindow(r.retryWindow))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy version, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Version was archived/deleted outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read policy version, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read policy version, no response body")
		return
	}

	// Archiving is a soft-delete: reads of an archived version may return 200
	// with archived_at set rather than 404.
	if isArchived(getResp.JSON200.ArchivedAt) {
		resp.State.RemoveResource(ctx)
		return
	}

	updatePolicyVersionComputedFromAPIResponse(getResp.JSON200, &data)

	// Preserve the configured cedar value to tolerate server-side normalization.
	// Only populate it from the response when unset (i.e. on import).
	if data.Cedar.IsNull() || data.Cedar.IsUnknown() {
		data.Cedar = nullableStringValue(getResp.JSON200.CedarRaw)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is unreachable: every attribute is RequiresReplace, so a change replaces
// the resource rather than updating it in place. Policy versions are immutable.
func (r *PolicyVersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Policy versions are immutable; this should never be called. Please report this issue to the provider developers.",
	)
}

func (r *PolicyVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyVersionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.client.ArchivePolicyVersionWithResponse(ctx, data.ZoneID.ValueString(), data.PolicyID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete policy version, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() == 400 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to archive policy version, got status 400: %s. This version is likely referenced by an active policy set binding; roll the set version/activation forward to a version that no longer includes it before destroying.", string(deleteResp.Body)),
		)
		return
	}

	if deleteResp.StatusCode() != 200 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete policy version, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *PolicyVersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zone_id/policy_id/version_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zone_id/policy_id/version_id', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// deprecatedSchemaWarning returns a warning message if the given schema version is
// deprecated, or an empty string otherwise (including when the lookup fails). This
// is best-effort: a single, un-retried lookup, so it stays silent (rather than
// blocking) when the zone is still bootstrapping or the schema does not exist.
func (r *PolicyVersionResource) deprecatedSchemaWarning(ctx context.Context, zoneID, schemaVersion string) string {
	schemaResp, err := r.client.GetPolicySchemaWithResponse(ctx, zoneID, schemaVersion, &client.GetPolicySchemaParams{})
	if err != nil || schemaResp.StatusCode() != 200 || schemaResp.JSON200 == nil {
		return ""
	}
	if schemaResp.JSON200.Status != client.SchemaVersionWithZoneInfoStatusDeprecated {
		return ""
	}
	return fmt.Sprintf("Schema version %q is deprecated. The policy version will still be created; consider pinning a current schema version.", schemaVersion)
}

// updatePolicyVersionComputedFromAPIResponse maps the computed and identity fields
// of a PolicyVersion API response onto the model. It deliberately leaves the cedar
// content untouched so the configured value is preserved across server-side
// normalization.
func updatePolicyVersionComputedFromAPIResponse(apiVersion *client.PolicyVersion, data *PolicyVersionResourceModel) {
	data.ID = types.StringValue(apiVersion.Id)
	data.ZoneID = types.StringValue(apiVersion.ZoneId)
	data.PolicyID = types.StringValue(apiVersion.PolicyId)
	data.SchemaVersion = types.StringValue(apiVersion.SchemaVersion)
	data.Version = types.Int64Value(int64(apiVersion.Version))
	data.Sha = types.StringValue(apiVersion.Sha)
	data.CreatedAt = types.StringValue(apiVersion.CreatedAt.Format(time.RFC3339))
	data.OwnerType = types.StringValue(string(apiVersion.OwnerType))
}
