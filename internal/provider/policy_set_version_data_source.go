package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PolicySetVersionDataSource{}

func NewPolicySetVersionDataSource(retryWindow time.Duration) datasource.DataSource {
	if retryWindow <= 0 {
		retryWindow = notFoundRetryWindow
	}
	return &PolicySetVersionDataSource{notFoundRetryWindow: retryWindow}
}

// PolicySetVersionDataSource defines the data source implementation.
type PolicySetVersionDataSource struct {
	client              *client.ClientWithResponses
	notFoundRetryWindow time.Duration
}

// PolicySetVersionDataSourceModel describes the data source data model. It is
// separate from the resource model: the data source surfaces archived versions
// instead of removing them.
type PolicySetVersionDataSourceModel struct {
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

func (d *PolicySetVersionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_set_version"
}

func (d *PolicySetVersionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an immutable policy set version by ID, including its manifest: the ordered list of `{policy_id, policy_version_id}` entries composing the set. Archived versions are still readable: `archived_at`/`archived_by` are populated rather than the read failing.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy set version.",
				Required:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this policy set version belongs to.",
				Required:            true,
			},
			"policy_set_id": schema.StringAttribute{
				MarkdownDescription: "The policy set this version is published under.",
				Required:            true,
			},
			"schema_version": schema.StringAttribute{
				MarkdownDescription: "The Cedar schema version pinned to this policy set version.",
				Computed:            true,
			},
			"manifest": schema.ListNestedAttribute{
				MarkdownDescription: "The ordered list of policy versions composing this set version.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"policy_id": schema.StringAttribute{
							MarkdownDescription: "The policy referenced by this entry.",
							Computed:            true,
						},
						"policy_version_id": schema.StringAttribute{
							MarkdownDescription: "The immutable policy version referenced by this entry.",
							Computed:            true,
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
				MarkdownDescription: "Whether this version is currently bound as the active policy set. Null when the API omits the binding status.",
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
				MarkdownDescription: "Timestamp when the version was archived (RFC 3339). Null unless archived.",
				Computed:            true,
			},
			"archived_by": schema.StringAttribute{
				MarkdownDescription: "Identifier of the actor that archived the version. Null unless archived.",
				Computed:            true,
			},
		},
	}
}

func (d *PolicySetVersionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *PolicySetVersionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicySetVersionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// svc-pdp is eventually consistent, so a version just created (or its zone
	// still provisioning) can 404 on a read replica briefly. A persistent 404
	// is a genuine miss, so bound retries to a short window.
	getResp, err := callWithRetry(ctx, func() (*client.GetPolicySetVersionResponse, error) {
		return d.client.GetPolicySetVersionWithResponse(ctx, data.ZoneID.ValueString(), data.PolicySetID.ValueString(), data.ID.ValueString())
	}, retryOnNotFound, withRetryWindow(d.notFoundRetryWindow))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy set version, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(
			"Policy Set Version Not Found",
			fmt.Sprintf("Policy set version with ID %s not found under policy set %s in zone %s", data.ID.ValueString(), data.PolicySetID.ValueString(), data.ZoneID.ValueString()),
		)
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

	// Archived versions read as 200 with archived_at set (intentional svc-pdp
	// semantics); surface them rather than erroring. zone_id is left as
	// configured since the response does not carry it.
	apiVersion := getResp.JSON200
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
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Manifest = manifestList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
