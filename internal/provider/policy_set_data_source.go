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
	_ datasource.DataSource                     = &PolicySetDataSource{}
	_ datasource.DataSourceWithConfigValidators = &PolicySetDataSource{}
)

func NewPolicySetDataSource(retryWindow time.Duration) datasource.DataSource {
	if retryWindow <= 0 {
		retryWindow = notFoundRetryWindow
	}
	return &PolicySetDataSource{notFoundRetryWindow: retryWindow}
}

// PolicySetDataSource defines the data source implementation.
type PolicySetDataSource struct {
	client              *client.ClientWithResponses
	notFoundRetryWindow time.Duration
}

func (d *PolicySetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_set"
}

func (d *PolicySetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Keycard policy set and its current binding status. A policy set is a container that binds policy set versions to a zone or user scope.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy set. Either `id` or `name` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this policy set belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the policy set. Either `id` or `name` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"target_type": schema.StringAttribute{
				MarkdownDescription: "What this policy set targets: `zone` or `user`.",
				Computed:            true,
			},
			"etag": schema.StringAttribute{
				MarkdownDescription: "Entity tag for optimistic concurrency control.",
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
			"latest_version_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the policy set's latest version. May be empty when the policy set has no versions.",
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
	}
}

func (d *PolicySetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PolicySetDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
	}
}

func (d *PolicySetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicySetModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var policySet *client.PolicySetWithBinding
	var etag types.String

	if !data.ID.IsNull() {
		// svc-pdp is eventually consistent, so a policy set just created (or its
		// zone still provisioning) can 404 on a read replica briefly. A
		// persistent 404 is a genuine miss, so bound retries to a short window.
		getResp, err := callWithRetry(ctx, func() (*client.GetPolicySetResponse, error) {
			return d.client.GetPolicySetWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
		}, retryOnNotFound, withRetryWindow(d.notFoundRetryWindow))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy set, got error: %s", err))
			return
		}

		if getResp.StatusCode() == 404 {
			resp.Diagnostics.AddError(
				"Policy Set Not Found",
				fmt.Sprintf("Policy set with ID %s not found in zone %s", data.ID.ValueString(), data.ZoneID.ValueString()),
			)
			return
		}

		if getResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read policy set, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
			)
			return
		}

		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to read policy set, no response body")
			return
		}

		policySet = getResp.JSON200
		etag = etagValue(getResp.HTTPResponse)
	} else {
		name := data.Name.ValueString()
		matches, err := findPolicySetsByName(ctx, d.client, data.ZoneID.ValueString(), name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policy sets: %s", err))
			return
		}

		if len(matches) == 0 {
			resp.Diagnostics.AddError(
				"Policy Set Not Found",
				fmt.Sprintf("No policy set found with name '%s' in zone '%s'", name, data.ZoneID.ValueString()),
			)
			return
		}

		if len(matches) > 1 {
			resp.Diagnostics.AddError(
				"Multiple Policy Sets Found",
				fmt.Sprintf("Expected exactly 1 policy set with name '%s' in zone '%s', but found %d. This indicates a data integrity issue.",
					name, data.ZoneID.ValueString(), len(matches)),
			)
			return
		}

		policySet = &matches[0]
		etag = types.StringNull()
	}

	updatePolicySetModelFromAPIResponse(policySet, &data)
	data.ETag = etag

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// findPolicySetsByName returns the policy sets in the zone whose name equals
// name exactly. query[name] is a case-insensitive substring match and the list
// is cursor-paginated, so it walks every page and filters client-side: an
// exact match can land beyond page 1 when other set names contain the
// requested name as a substring. Stops early once two matches are found, since
// callers treat that as an error.
func findPolicySetsByName(ctx context.Context, c *client.ClientWithResponses, zoneID, name string) ([]client.PolicySetWithBinding, error) {
	var matches []client.PolicySetWithBinding
	var after *string
	for {
		// A zone is provisioned asynchronously, so the list endpoint may 404
		// ("zone not found") briefly after the zone is created. That is the
		// only 404 this endpoint returns — a missing name is an empty list,
		// not a 404 — so unlike the ID lookup there is no genuine miss to
		// surface quickly, and the full default retry window applies.
		listResp, err := callWithRetry(ctx, func() (*client.ListPolicySetsResponse, error) {
			return c.ListPolicySetsWithResponse(ctx, zoneID, &client.ListPolicySetsParams{
				QueryName: &[]string{name},
				After:     after,
			})
		}, retryOnNotFound)
		if err != nil {
			return nil, err
		}

		if listResp.JSON200 == nil {
			return nil, fmt.Errorf("received empty response from API (status %d)", listResp.StatusCode())
		}

		for _, ps := range listResp.JSON200.Items {
			if ps.Name == name {
				matches = append(matches, ps)
			}
		}

		if len(matches) > 1 {
			return matches, nil
		}
		// A null or absent after_cursor means no next page.
		cursor := nullableStringValue(listResp.JSON200.Pagination.AfterCursor)
		if cursor.IsNull() || cursor.ValueString() == "" {
			return matches, nil
		}
		after = cursor.ValueStringPointer()
	}
}
