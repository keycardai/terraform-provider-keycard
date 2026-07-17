package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PolicySetsDataSource{}

func NewPolicySetsDataSource() datasource.DataSource {
	return &PolicySetsDataSource{}
}

// PolicySetsDataSource defines the data source implementation.
type PolicySetsDataSource struct {
	client *client.ClientWithResponses
}

// PolicySetsDataSourceModel describes the data source data model.
type PolicySetsDataSourceModel struct {
	ZoneID     types.String `tfsdk:"zone_id"`
	Name       types.String `tfsdk:"name"`
	PolicySets types.List   `tfsdk:"policy_sets"`
}

// policySetsEntryModel describes one policy set in the list. It mirrors the
// singular data source's attributes minus etag, which is carried in a response
// header the list endpoint does not provide per item.
type policySetsEntryModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	TargetType      types.String `tfsdk:"target_type"`
	OwnerType       types.String `tfsdk:"owner_type"`
	CreatedBy       types.String `tfsdk:"created_by"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	UpdatedBy       types.String `tfsdk:"updated_by"`
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

func policySetsEntryAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                types.StringType,
		"name":              types.StringType,
		"target_type":       types.StringType,
		"owner_type":        types.StringType,
		"created_by":        types.StringType,
		"created_at":        types.StringType,
		"updated_at":        types.StringType,
		"updated_by":        types.StringType,
		"latest_version_id": types.StringType,
		"latest_version":    types.Int64Type,
		"active":            types.BoolType,
		"active_version_id": types.StringType,
		"active_version":    types.Int64Type,
		"shadow_version_id": types.StringType,
		"shadow_version":    types.Int64Type,
		"target_id":         types.StringType,
		"mode":              types.StringType,
	}
}

func (d *PolicySetsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_sets"
}

func (d *PolicySetsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the policy sets in a zone with their current binding status, optionally filtered by name. Archived policy sets are excluded.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone to list policy sets from.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Optional name filter, applied server-side as a case-insensitive substring match.",
				Optional:            true,
			},
			"policy_sets": schema.ListNestedAttribute{
				MarkdownDescription: "The policy sets matching the filter, or all policy sets in the zone when no filter is set.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier of the policy set.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Human-readable name of the policy set.",
							Computed:            true,
						},
						"target_type": schema.StringAttribute{
							MarkdownDescription: "What this policy set targets: `zone` or `user`.",
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
						"updated_by": schema.StringAttribute{
							MarkdownDescription: "Identifier of the actor that last updated the policy set. Null when never updated.",
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
							MarkdownDescription: "Target entity ID of the active binding. Null when unbound.",
							Computed:            true,
						},
						"mode": schema.StringAttribute{
							MarkdownDescription: "Binding mode of the active binding: `active` or `shadow`. Null when unbound.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *PolicySetsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PolicySetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicySetsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var queryName *[]string
	if !data.Name.IsNull() {
		queryName = &[]string{data.Name.ValueString()}
	}

	policySets, err := listAllPages(func(after *string) ([]client.PolicySetWithBinding, nullable.Nullable[string], error) {
		// A zone is provisioned asynchronously, so the list endpoint may 404
		// ("zone not found") briefly after the zone is created. That is the
		// only 404 this endpoint returns, so the full default retry window
		// applies.
		listResp, err := callWithRetry(ctx, func() (*client.ListPolicySetsResponse, error) {
			return d.client.ListPolicySetsWithResponse(ctx, data.ZoneID.ValueString(), &client.ListPolicySetsParams{
				QueryName: queryName,
				After:     after,
			})
		}, retryOnNotFound)
		if err != nil {
			return nil, nullable.Nullable[string]{}, err
		}
		if listResp.JSON200 == nil {
			return nil, nullable.Nullable[string]{}, fmt.Errorf("received empty response from API (status %d): %s", listResp.StatusCode(), string(listResp.Body))
		}
		return listResp.JSON200.Items, listResp.JSON200.Pagination.AfterCursor, nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policy sets: %s", err))
		return
	}

	entries := make([]policySetsEntryModel, len(policySets))
	for i, ps := range policySets {
		mode := types.StringNull()
		if m, err := ps.Mode.Get(); err == nil {
			mode = types.StringValue(string(m))
		}
		entries[i] = policySetsEntryModel{
			ID:              types.StringValue(ps.Id),
			Name:            types.StringValue(ps.Name),
			TargetType:      types.StringValue(string(ps.TargetType)),
			OwnerType:       types.StringValue(string(ps.OwnerType)),
			CreatedBy:       types.StringValue(ps.CreatedBy),
			CreatedAt:       types.StringValue(ps.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:       types.StringValue(ps.UpdatedAt.Format(time.RFC3339)),
			UpdatedBy:       nullableStringValue(ps.UpdatedBy),
			LatestVersionID: nullableStringValue(ps.LatestVersionId),
			LatestVersion:   nullableInt64Value(ps.LatestVersion),
			Active:          types.BoolValue(ps.Active),
			ActiveVersionID: nullableStringValue(ps.ActiveVersionId),
			ActiveVersion:   nullableInt64Value(ps.ActiveVersion),
			ShadowVersionID: nullableStringValue(ps.ShadowVersionId),
			ShadowVersion:   nullableInt64Value(ps.ShadowVersion),
			TargetID:        nullableStringValue(ps.TargetId),
			Mode:            mode,
		}
	}

	policySetsList, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: policySetsEntryAttrTypes()}, entries)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.PolicySets = policySetsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
