package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &PolicySetVersionResource{}
	_ resource.ResourceWithImportState = &PolicySetVersionResource{}
)

// NewPolicySetVersionResource builds the resource with the given Read retry
// window. Zero selects the full apiRetryWindow: a false 404 on a resource Read
// removes the resource from state and forces a destructive recreate, so
// production must wait out replica lag. Tests exercising a genuine miss inject a
// short window.
func NewPolicySetVersionResource(retryWindow time.Duration) resource.Resource {
	if retryWindow <= 0 {
		retryWindow = apiRetryWindow
	}
	return &PolicySetVersionResource{retryWindow: retryWindow}
}

// PolicySetVersionResource defines the resource implementation.
type PolicySetVersionResource struct {
	client      *client.ClientWithResponses
	retryWindow time.Duration
}

// PolicySetVersionResourceModel describes the policy set version data model.
type PolicySetVersionResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ZoneID        types.String `tfsdk:"zone_id"`
	PolicySetID   types.String `tfsdk:"policy_set_id"`
	SchemaVersion types.String `tfsdk:"schema_version"`
	Manifest      types.List   `tfsdk:"manifest"`
	Version       types.Int64  `tfsdk:"version"`
	ManifestSha   types.String `tfsdk:"manifest_sha"`
	Active        types.Bool   `tfsdk:"active"`
	CreatedAt     types.String `tfsdk:"created_at"`
	CreatedBy     types.String `tfsdk:"created_by"`
	OwnerType     types.String `tfsdk:"owner_type"`
	ArchivedAt    types.String `tfsdk:"archived_at"`
	ArchivedBy    types.String `tfsdk:"archived_by"`
}

// policySetVersionManifestEntryModel describes one manifest entry.
type policySetVersionManifestEntryModel struct {
	PolicyID        types.String `tfsdk:"policy_id"`
	PolicyVersionID types.String `tfsdk:"policy_version_id"`
	Sha             types.String `tfsdk:"sha"`
}

func manifestEntryAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"policy_id":         types.StringType,
		"policy_version_id": types.StringType,
		"sha":               types.StringType,
	}
}

func (r *PolicySetVersionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_set_version"
}

func (r *PolicySetVersionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Publishes an immutable manifest snapshot of a `keycard_policy_set`: an ordered list of `{policy_id, policy_version_id}` entries pinned to a `schema_version`. Any change replaces the resource. Use `lifecycle { create_before_destroy = true }` so the new version is published before the old one is archived. This resource does not activate the version; use `keycard_policy_set_activation` for that.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy set version.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this policy set version belongs to. Changing this creates a new version.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_set_id": schema.StringAttribute{
				MarkdownDescription: "The policy set this version is published under. Changing this creates a new version.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema_version": schema.StringAttribute{
				MarkdownDescription: "The Cedar schema version pinned to this policy set version. Every referenced policy version must be validated against this schema. Changing it publishes a new version.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"manifest": schema.ListNestedAttribute{
				MarkdownDescription: "The ordered list of policy versions composing this set version. Immutable: changing any entry publishes a new version. Each referenced policy version must not be archived and must be validated against the same `schema_version`.",
				Required:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"policy_id": schema.StringAttribute{
							MarkdownDescription: "The policy referenced by this entry.",
							Required:            true,
						},
						"policy_version_id": schema.StringAttribute{
							MarkdownDescription: "The immutable policy version referenced by this entry.",
							Required:            true,
						},
						"sha": schema.StringAttribute{
							MarkdownDescription: "Hex-encoded hash of the referenced policy version content, computed by the server.",
							Computed:            true,
						},
					},
				},
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "Human-readable version number, incremented per published version (e.g., 1, 2, 3).",
				Computed:            true,
			},
			"manifest_sha": schema.StringAttribute{
				MarkdownDescription: "Hex-encoded hash of the canonicalized manifest.",
				Computed:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether this version is currently bound as the active policy set. Managed by `keycard_policy_set_activation`, not this resource.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the version was created (RFC 3339).",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Identifier of the actor that created the version.",
				Computed:            true,
			},
			"owner_type": schema.StringAttribute{
				MarkdownDescription: "Who manages this policy set version: `platform` (managed by Keycard) or `customer` (managed by the tenant).",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the version was archived (RFC 3339). Always null while the version is managed by Terraform: an out-of-band archive removes the resource from state.",
				Computed:            true,
			},
			"archived_by": schema.StringAttribute{
				MarkdownDescription: "Identifier of the actor that archived the version. Null unless archived.",
				Computed:            true,
			},
		},
	}
}

func (r *PolicySetVersionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicySetVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicySetVersionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var entries []policySetVersionManifestEntryModel
	resp.Diagnostics.Append(data.Manifest.ElementsAs(ctx, &entries, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiEntries := make([]client.PolicySetManifestEntry, len(entries))
	for i, e := range entries {
		apiEntries[i] = client.PolicySetManifestEntry{
			PolicyId:        e.PolicyID.ValueString(),
			PolicyVersionId: e.PolicyVersionID.ValueString(),
		}
	}

	// The server silently accepts deprecated schemas; warn at apply time so
	// pinning a superseded schema is visible. Best-effort: a lookup failure is
	// not fatal to the create.
	if warning := deprecatedSchemaWarning(ctx, r.client, data.ZoneID.ValueString(), data.SchemaVersion.ValueString()); warning != "" {
		resp.Diagnostics.AddWarning("Deprecated Policy Schema", warning)
	}

	createReq := client.CreatePolicySetVersionRequest{
		Manifest:      client.PolicySetManifest{Entries: apiEntries},
		SchemaVersion: data.SchemaVersion.ValueString(),
	}

	// A zone is provisioned asynchronously across services, so the endpoint may
	// 404 ("zone not found") briefly after the zone is created.
	createResp, err := callWithRetry(ctx, func() (*client.CreatePolicySetVersionResponse, error) {
		return r.client.CreatePolicySetVersionWithResponse(ctx, data.ZoneID.ValueString(), data.PolicySetID.ValueString(), createReq)
	}, retryOnNotFound)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy set version, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create policy set version, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create policy set version, no response body")
		return
	}

	resp.Diagnostics.Append(updatePolicySetVersionComputedFromAPIResponse(ctx, createResp.JSON201, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicySetVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicySetVersionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// svc-pdp is eventually consistent: a version just created (or its zone still
	// provisioning) can 404 on a read replica for a while. Retry transient 404s
	// over the configured window; acting on a false 404 here removes the resource
	// from state and forces a destructive recreate, so production waits out
	// replica lag rather than surfacing a genuine miss quickly.
	getResp, err := callWithRetry(ctx, func() (*client.GetPolicySetVersionResponse, error) {
		return r.client.GetPolicySetVersionWithResponse(ctx, data.ZoneID.ValueString(), data.PolicySetID.ValueString(), data.ID.ValueString())
	}, retryOnNotFound, withRetryWindow(r.retryWindow))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy set version, got error: %s", err))
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
			fmt.Sprintf("Unable to read policy set version, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read policy set version, no response body")
		return
	}

	// Archiving is a soft-delete: reads of an archived version return 200 with
	// archived_at set, not 404 (the GET query only filters the parent set's
	// archived_at), so this is the drift path for out-of-band archives.
	if isArchived(getResp.JSON200.ArchivedAt) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(updatePolicySetVersionComputedFromAPIResponse(ctx, getResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is unreachable: every configurable attribute is RequiresReplace, so a
// change replaces the resource rather than updating it in place. Policy set
// versions are immutable.
func (r *PolicySetVersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Policy set versions are immutable; this should never be called. Please report this issue to the provider developers.",
	)
}

func (r *PolicySetVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicySetVersionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No 404 retry here, unlike Create: a 404 from archive means "already
	// archived" and is accepted as success below, and the provisioning lag
	// Create must survive has long converged by the time a managed resource is
	// destroyed. Retrying would make every already-gone destroy wait out the
	// window without being able to tell the two apart. Same shape as the other
	// archive-backed Deletes.
	deleteResp, err := r.client.ArchivePolicySetVersionWithResponse(ctx, data.ZoneID.ValueString(), data.PolicySetID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive policy set version, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() == 400 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to archive policy set version, got status 400: %s. This version is currently bound (active or shadow); roll the activation forward to another version or remove the shadow binding before destroying.", string(deleteResp.Body)),
		)
		return
	}

	if deleteResp.StatusCode() != 200 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to archive policy set version, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *PolicySetVersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zone_id/policy_set_id/version_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zone_id/policy_set_id/version_id', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_set_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// updatePolicySetVersionComputedFromAPIResponse maps the computed and identity
// fields of a PolicySetVersion API response onto the model, including rebuilding
// the manifest list so per-entry shas populate. zone_id is left untouched since
// the response does not carry it.
func updatePolicySetVersionComputedFromAPIResponse(ctx context.Context, apiVersion *client.PolicySetVersion, data *PolicySetVersionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(apiVersion.Id)
	data.PolicySetID = types.StringValue(apiVersion.PolicySetId)
	data.SchemaVersion = types.StringValue(apiVersion.SchemaVersion)
	data.Version = types.Int64Value(int64(apiVersion.Version))
	data.ManifestSha = types.StringValue(apiVersion.ManifestSha)
	data.CreatedAt = types.StringValue(apiVersion.CreatedAt.Format(time.RFC3339))
	data.CreatedBy = types.StringValue(apiVersion.CreatedBy)
	data.OwnerType = types.StringValue(string(apiVersion.OwnerType))
	data.ArchivedAt = nullableTimeValue(apiVersion.ArchivedAt)
	data.ArchivedBy = nullableStringValue(apiVersion.ArchivedBy)

	// nil means the API omitted the field, not "inactive" — surface unknown as
	// null rather than fabricating false.
	data.Active = types.BoolPointerValue(apiVersion.Active)

	entries := make([]policySetVersionManifestEntryModel, len(apiVersion.Manifest.Entries))
	for i, e := range apiVersion.Manifest.Entries {
		sha := types.StringNull()
		if e.Sha != nil {
			sha = types.StringValue(*e.Sha)
		}
		entries[i] = policySetVersionManifestEntryModel{
			PolicyID:        types.StringValue(e.PolicyId),
			PolicyVersionID: types.StringValue(e.PolicyVersionId),
			Sha:             sha,
		}
	}

	manifestList, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: manifestEntryAttrTypes()}, entries)
	diags.Append(listDiags...)
	data.Manifest = manifestList

	return diags
}
