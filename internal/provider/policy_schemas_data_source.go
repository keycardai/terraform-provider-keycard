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
var _ datasource.DataSource = &PolicySchemasDataSource{}

func NewPolicySchemasDataSource() datasource.DataSource {
	return &PolicySchemasDataSource{}
}

// PolicySchemasDataSource defines the data source implementation.
type PolicySchemasDataSource struct {
	client *client.ClientWithResponses
}

// PolicySchemasDataSourceModel describes the data source data model.
type PolicySchemasDataSourceModel struct {
	ZoneID    types.String `tfsdk:"zone_id"`
	IsDefault types.Bool   `tfsdk:"is_default"`
	Schemas   types.List   `tfsdk:"schemas"`
}

// policySchemasEntryModel describes one schema version in the list.
type policySchemasEntryModel struct {
	Version      types.String `tfsdk:"version"`
	Status       types.String `tfsdk:"status"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
	CedarSchema  types.String `tfsdk:"cedar_schema"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
	DeprecatedAt types.String `tfsdk:"deprecated_at"`
	ArchivedAt   types.String `tfsdk:"archived_at"`
}

func policySchemasEntryAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"version":       types.StringType,
		"status":        types.StringType,
		"is_default":    types.BoolType,
		"cedar_schema":  types.StringType,
		"created_at":    types.StringType,
		"updated_at":    types.StringType,
		"deprecated_at": types.StringType,
		"archived_at":   types.StringType,
	}
}

func (d *PolicySchemasDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_schemas"
}

func (d *PolicySchemasDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the Cedar policy schema versions available in a zone. Policy schemas define the entity model (users, applications, resources) that Cedar policies are validated against; they are managed by the Keycard platform. Unlike `keycard_policies` and `keycard_policy_sets`, archived schema versions are included; filter client-side on `status != \"archived\"` if you want only active ones.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone to list policy schemas from.",
				Required:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Optional filter by default status. When `true`, returns only the zone's default schema; when `false`, only non-default schemas. Omit to return all schemas.",
				Optional:            true,
			},
			"schemas": schema.ListNestedAttribute{
				MarkdownDescription: "The schema versions matching the filter, or all schema versions in the zone when no filter is set.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							MarkdownDescription: "Schema version string (e.g. `2026-02-24`).",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Lifecycle status of the schema version: `active`, `deprecated`, or `archived`. New policy versions cannot be created against archived schemas.",
							Computed:            true,
						},
						"is_default": schema.BoolAttribute{
							MarkdownDescription: "Whether this schema version is the zone's default.",
							Computed:            true,
						},
						"cedar_schema": schema.StringAttribute{
							MarkdownDescription: "The Cedar schema in human-readable Cedar syntax.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Timestamp when the schema version was created (RFC 3339).",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "Timestamp when the schema version was last updated (RFC 3339).",
							Computed:            true,
						},
						"deprecated_at": schema.StringAttribute{
							MarkdownDescription: "Timestamp when the schema version was deprecated (RFC 3339). Null unless deprecated.",
							Computed:            true,
						},
						"archived_at": schema.StringAttribute{
							MarkdownDescription: "Timestamp when the schema version was archived (RFC 3339). Null unless archived.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *PolicySchemasDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PolicySchemasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicySchemasDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	format := client.ListPolicySchemasParamsFormatCedar
	var isDefault *bool
	if !data.IsDefault.IsNull() {
		isDefault = data.IsDefault.ValueBoolPointer()
	}

	schemas, err := listAllPages(func(after *string) ([]client.SchemaVersionWithZoneInfo, nullable.Nullable[string], error) {
		// A 404 here can only mean the zone hasn't finished provisioning, so
		// the full default retry window applies.
		listResp, err := callWithRetry(ctx, func() (*client.ListPolicySchemasResponse, error) {
			return d.client.ListPolicySchemasWithResponse(ctx, data.ZoneID.ValueString(), &client.ListPolicySchemasParams{
				Format:    &format,
				IsDefault: isDefault,
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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policy schemas: %s", err))
		return
	}

	entries := make([]policySchemasEntryModel, len(schemas))
	for i, s := range schemas {
		entries[i] = policySchemasEntryModel{
			Version:      types.StringValue(s.Version),
			Status:       types.StringValue(string(s.Status)),
			IsDefault:    types.BoolValue(s.IsDefault),
			CedarSchema:  nullableStringValue(s.CedarSchema),
			CreatedAt:    types.StringValue(s.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:    types.StringValue(s.UpdatedAt.Format(time.RFC3339)),
			DeprecatedAt: nullableTimeValue(s.DeprecatedAt),
			ArchivedAt:   nullableTimeValue(s.ArchivedAt),
		}
	}

	schemasList, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: policySchemasEntryAttrTypes()}, entries)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Schemas = schemasList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
