package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource                     = &GroupDataSource{}
	_ datasource.DataSourceWithConfigValidators = &GroupDataSource{}
)

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

// GroupDataSource defines the data source implementation.
type GroupDataSource struct {
	client *client.ClientWithResponses
}

func (d *GroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Keycard group. A group is a zone-scoped collection of users that can be assigned roles and referenced in policies.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the group. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this group belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the group.",
				Computed:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "User-specified identifier for the group, unique within the zone. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (d *GroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("identifier"),
		),
	}
}

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var group *client.Group

	if !data.ID.IsNull() {
		// Lookup by ID
		getResp, err := d.client.GetGroupWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString(), nil)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group, got error: %s", err))
			return
		}

		if getResp.StatusCode() == 404 {
			resp.Diagnostics.AddError(
				"Group Not Found",
				fmt.Sprintf("Group with ID %s not found in zone %s", data.ID.ValueString(), data.ZoneID.ValueString()),
			)
			return
		}

		if getResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read group, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
			)
			return
		}

		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to read group, no response body")
			return
		}

		group = getResp.JSON200
	} else {
		// Lookup by identifier
		identifier := data.Identifier.ValueString()
		listResp, err := d.client.ListGroupsWithResponse(ctx, data.ZoneID.ValueString(), &client.ListGroupsParams{
			FilterIdentifier: &[]string{identifier},
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list groups: %s", err))
			return
		}

		if listResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to list groups, got status %d: %s", listResp.StatusCode(), string(listResp.Body)),
			)
			return
		}

		if listResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Received empty response from API")
			return
		}

		resultCount := len(listResp.JSON200.Items)
		if resultCount == 0 {
			resp.Diagnostics.AddError(
				"Group Not Found",
				fmt.Sprintf("No group found with identifier '%s' in zone '%s'", identifier, data.ZoneID.ValueString()),
			)
			return
		}

		if resultCount > 1 {
			resp.Diagnostics.AddError(
				"Multiple Groups Found",
				fmt.Sprintf("Expected exactly 1 group with identifier '%s' in zone '%s', but found %d. This indicates a data integrity issue.",
					identifier, data.ZoneID.ValueString(), resultCount),
			)
			return
		}

		group = &listResp.JSON200.Items[0]
	}

	updateGroupModelFromAPIResponse(group, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
