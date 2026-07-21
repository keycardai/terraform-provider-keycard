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
var _ datasource.DataSource = &PolicyVersionDataSource{}

func NewPolicyVersionDataSource(retryWindow time.Duration) datasource.DataSource {
	if retryWindow <= 0 {
		retryWindow = notFoundRetryWindow
	}
	return &PolicyVersionDataSource{notFoundRetryWindow: retryWindow}
}

// PolicyVersionDataSource defines the data source implementation.
type PolicyVersionDataSource struct {
	client              *client.ClientWithResponses
	notFoundRetryWindow time.Duration
}

// PolicyVersionDataSourceModel describes the data source data model. It is
// separate from the resource model: the data source reports the
// server-normalized cedar content verbatim rather than preserving a configured
// value, and surfaces archived versions instead of removing them.
type PolicyVersionDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	ZoneID        types.String `tfsdk:"zone_id"`
	PolicyID      types.String `tfsdk:"policy_id"`
	Cedar         types.String `tfsdk:"cedar"`
	SchemaVersion types.String `tfsdk:"schema_version"`
	Version       types.Int64  `tfsdk:"version"`
	Sha           types.String `tfsdk:"sha"`
	CreatedAt     types.String `tfsdk:"created_at"`
	CreatedBy     types.String `tfsdk:"created_by"`
	OwnerType     types.String `tfsdk:"owner_type"`
	ArchivedAt    types.String `tfsdk:"archived_at"`
	ArchivedBy    types.String `tfsdk:"archived_by"`
}

func (d *PolicyVersionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_version"
}

func (d *PolicyVersionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an immutable Cedar policy version by ID. Archived versions are still readable: `archived_at`/`archived_by` are populated rather than the read failing. The `cedar` content is the server-normalized form, which may differ in formatting from what was originally published.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy version.",
				Required:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this policy version belongs to.",
				Required:            true,
			},
			"policy_id": schema.StringAttribute{
				MarkdownDescription: "The policy this version is published under.",
				Required:            true,
			},
			"cedar": schema.StringAttribute{
				MarkdownDescription: "The Cedar policy content in human-readable syntax, as normalized and stored by the server.",
				Computed:            true,
			},
			"schema_version": schema.StringAttribute{
				MarkdownDescription: "The Cedar schema version this policy was validated against.",
				Computed:            true,
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
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Identifier of the actor that created the version.",
				Computed:            true,
			},
			"owner_type": schema.StringAttribute{
				MarkdownDescription: "Who manages this policy version: `platform` (managed by Keycard) or `customer` (managed by the tenant).",
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

func (d *PolicyVersionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PolicyVersionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicyVersionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// svc-pdp is eventually consistent, so a version just created (or its zone
	// still provisioning) can 404 on a read replica briefly. A persistent 404
	// is a genuine miss, so bound retries to a short window.
	getResp, err := callWithRetry(ctx, func() (*client.GetPolicyVersionResponse, error) {
		return d.client.GetPolicyVersionWithResponse(ctx, data.ZoneID.ValueString(), data.PolicyID.ValueString(), data.ID.ValueString())
	}, retryOnNotFound, withRetryWindow(d.notFoundRetryWindow))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy version, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(
			"Policy Version Not Found",
			fmt.Sprintf("Policy version with ID %s not found under policy %s in zone %s", data.ID.ValueString(), data.PolicyID.ValueString(), data.ZoneID.ValueString()),
		)
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

	// Archived versions read as 200 with archived_at set (intentional svc-pdp
	// semantics); surface them rather than erroring.
	apiVersion := getResp.JSON200
	data.ID = types.StringValue(apiVersion.Id)
	data.ZoneID = types.StringValue(apiVersion.ZoneId)
	data.PolicyID = types.StringValue(apiVersion.PolicyId)
	data.Cedar = nullableStringValue(apiVersion.CedarRaw)
	data.SchemaVersion = types.StringValue(apiVersion.SchemaVersion)
	data.Version = types.Int64Value(int64(apiVersion.Version))
	data.Sha = types.StringValue(apiVersion.Sha)
	data.CreatedAt = types.StringValue(apiVersion.CreatedAt.Format(time.RFC3339))
	data.CreatedBy = types.StringValue(apiVersion.CreatedBy)
	data.OwnerType = types.StringValue(string(apiVersion.OwnerType))
	data.ArchivedAt = nullableTimeValue(apiVersion.ArchivedAt)
	data.ArchivedBy = nullableStringValue(apiVersion.ArchivedBy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
