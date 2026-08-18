package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource                     = &RoleDataSource{}
	_ datasource.DataSourceWithConfigValidators = &RoleDataSource{}
)

func NewRoleDataSource(retryWindow time.Duration) datasource.DataSource {
	if retryWindow <= 0 {
		retryWindow = notFoundRetryWindow
	}
	return &RoleDataSource{notFoundRetryWindow: retryWindow}
}

// RoleDataSource defines the data source implementation.
type RoleDataSource struct {
	client              *client.ClientWithResponses
	notFoundRetryWindow time.Duration
}

// RoleModel maps the keycard_role data source schema.
type RoleModel struct {
	ID          types.String `tfsdk:"id"`
	ZoneID      types.String `tfsdk:"zone_id"`
	Identifier  types.String `tfsdk:"identifier"`
	OwnerType   types.String `tfsdk:"owner_type"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *RoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Keycard role in a zone. Look the role up either by `id`, or by `identifier` together with `owner_type`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the role. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this role belongs to.",
				Required:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Role identifier: a lowercase slug, unique per owner type within a zone. Requires `owner_type`. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"owner_type": schema.StringAttribute{
				MarkdownDescription: "Who owns this role: `platform` (managed by Keycard) or `customer` (managed by the tenant). Required when looking the role up by `identifier`, since an identifier is only unique per owner type. Must be omitted when `id` is set.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Human-readable description of the role. Null when unset.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the role was created (RFC 3339).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the role was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (d *RoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoleDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("identifier"),
		),
		datasourcevalidator.Conflicting(
			path.MatchRoot("id"),
			path.MatchRoot("identifier"),
		),
		datasourcevalidator.Conflicting(
			path.MatchRoot("id"),
			path.MatchRoot("owner_type"),
		),
		datasourcevalidator.RequiredTogether(
			path.MatchRoot("identifier"),
			path.MatchRoot("owner_type"),
		),
	}
}

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()

	var role *client.Role

	if !data.ID.IsNull() {
		// A zone is provisioned asynchronously, so a role read shortly after the
		// zone is created can 404 briefly. A persistent 404 is a genuine miss,
		// so bound retries to a short window.
		getResp, err := callWithRetry(ctx, func() (*client.GetRoleResponse, error) {
			return d.client.GetRoleWithResponse(ctx, zoneID, data.ID.ValueString())
		}, retryOnNotFound, withRetryWindow(d.notFoundRetryWindow))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role, got error: %s", err))
			return
		}

		if getResp.StatusCode() == 404 {
			resp.Diagnostics.AddError(
				"Role Not Found",
				fmt.Sprintf("Role with ID %s not found in zone %s", data.ID.ValueString(), zoneID),
			)
			return
		}

		if getResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read role, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
			)
			return
		}

		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to read role, no response body")
			return
		}

		role = getResp.JSON200
	} else {
		identifier := data.Identifier.ValueString()
		ownerType := data.OwnerType.ValueString()

		// The identifier filter returns at most two roles — one per owner type —
		// so a single unpaginated call suffices. As with the ID lookup, the zone
		// itself may 404 briefly while it provisions; a missing identifier is an
		// empty list rather than a 404, so the full default retry window applies.
		listResp, err := callWithRetry(ctx, func() (*client.ListRolesResponse, error) {
			return d.client.ListRolesWithResponse(ctx, zoneID, &client.ListRolesParams{
				Identifier: &identifier,
			})
		}, retryOnNotFound)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles: %s", err))
			return
		}

		if listResp.JSON200 == nil {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to list roles, got status %d: %s", listResp.StatusCode(), string(listResp.Body)),
			)
			return
		}

		for i, r := range listResp.JSON200.Items {
			if r.Identifier == identifier && string(r.OwnerType) == ownerType {
				role = &listResp.JSON200.Items[i]
				break
			}
		}

		if role == nil {
			resp.Diagnostics.AddError(
				"Role Not Found",
				fmt.Sprintf("No %s role found with identifier '%s' in zone '%s'", ownerType, identifier, zoneID),
			)
			return
		}
	}

	data.ID = types.StringValue(role.Id)
	data.ZoneID = types.StringValue(role.ZoneId)
	data.Identifier = types.StringValue(role.Identifier)
	data.OwnerType = types.StringValue(string(role.OwnerType))
	data.Description = nullableStringValue(role.Description)
	data.CreatedAt = types.StringValue(role.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(role.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
