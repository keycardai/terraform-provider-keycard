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
var _ datasource.DataSource = &PoliciesDataSource{}

func NewPoliciesDataSource() datasource.DataSource {
	return &PoliciesDataSource{}
}

// PoliciesDataSource defines the data source implementation.
type PoliciesDataSource struct {
	client *client.ClientWithResponses
}

// PoliciesDataSourceModel describes the data source data model.
type PoliciesDataSourceModel struct {
	ZoneID   types.String `tfsdk:"zone_id"`
	Name     types.String `tfsdk:"name"`
	Policies types.List   `tfsdk:"policies"`
}

// policiesEntryModel describes one policy in the list.
type policiesEntryModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	OwnerType           types.String `tfsdk:"owner_type"`
	CreatedBy           types.String `tfsdk:"created_by"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	UpdatedBy           types.String `tfsdk:"updated_by"`
	LatestVersionID     types.String `tfsdk:"latest_version_id"`
	LatestVersion       types.Int64  `tfsdk:"latest_version"`
	LatestSchemaVersion types.String `tfsdk:"latest_schema_version"`
}

func policiesEntryAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                    types.StringType,
		"name":                  types.StringType,
		"description":           types.StringType,
		"owner_type":            types.StringType,
		"created_by":            types.StringType,
		"created_at":            types.StringType,
		"updated_at":            types.StringType,
		"updated_by":            types.StringType,
		"latest_version_id":     types.StringType,
		"latest_version":        types.Int64Type,
		"latest_schema_version": types.StringType,
	}
}

func (d *PoliciesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policies"
}

func (d *PoliciesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the policies in a zone, optionally filtered by name. Archived policies are excluded.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone to list policies from.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Optional name filter, applied server-side as a case-insensitive substring match.",
				Optional:            true,
			},
			"policies": schema.ListNestedAttribute{
				MarkdownDescription: "The policies matching the filter, or all policies in the zone when no filter is set.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier of the policy.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Human-readable name of the policy.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Optional description of the policy's purpose. May be empty.",
							Computed:            true,
						},
						"owner_type": schema.StringAttribute{
							MarkdownDescription: "Who manages this policy: `platform` (managed by Keycard) or `customer` (managed by the tenant).",
							Computed:            true,
						},
						"created_by": schema.StringAttribute{
							MarkdownDescription: "Identifier of the actor that created the policy.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Timestamp when the policy was created (RFC 3339).",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "Timestamp when the policy was last updated (RFC 3339).",
							Computed:            true,
						},
						"updated_by": schema.StringAttribute{
							MarkdownDescription: "Identifier of the actor that last updated the policy. Null when never updated.",
							Computed:            true,
						},
						"latest_version_id": schema.StringAttribute{
							MarkdownDescription: "Identifier of the policy's latest version. Null when the policy has no published versions.",
							Computed:            true,
						},
						"latest_version": schema.Int64Attribute{
							MarkdownDescription: "Human-readable version number of the latest version. Null when the policy has no published versions.",
							Computed:            true,
						},
						"latest_schema_version": schema.StringAttribute{
							MarkdownDescription: "Schema version the latest version was validated against. Null when the policy has no published versions.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *PoliciesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PoliciesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var queryName *[]string
	if !data.Name.IsNull() {
		queryName = &[]string{data.Name.ValueString()}
	}

	policies, err := listAllPages(func(after *string) ([]client.Policy, nullable.Nullable[string], error) {
		// A zone is provisioned asynchronously, so the list endpoint may 404
		// ("zone not found") briefly after the zone is created. That is the
		// only 404 this endpoint returns, so the full default retry window
		// applies.
		listResp, err := callWithRetry(ctx, func() (*client.ListPoliciesResponse, error) {
			return d.client.ListPoliciesWithResponse(ctx, data.ZoneID.ValueString(), &client.ListPoliciesParams{
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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policies: %s", err))
		return
	}

	entries := make([]policiesEntryModel, len(policies))
	for i, p := range policies {
		entries[i] = policiesEntryModel{
			ID:                  types.StringValue(p.Id),
			Name:                types.StringValue(p.Name),
			Description:         nullableStringValue(p.Description),
			OwnerType:           types.StringValue(string(p.OwnerType)),
			CreatedBy:           types.StringValue(p.CreatedBy),
			CreatedAt:           types.StringValue(p.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:           types.StringValue(p.UpdatedAt.Format(time.RFC3339)),
			UpdatedBy:           nullableStringValue(p.UpdatedBy),
			LatestVersionID:     nullableStringValue(p.LatestVersionId),
			LatestVersion:       nullableInt64Value(p.LatestVersion),
			LatestSchemaVersion: nullableStringValue(p.LatestSchemaVersion),
		}
	}

	policiesList, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: policiesEntryAttrTypes()}, entries)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Policies = policiesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
