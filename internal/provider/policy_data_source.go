package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource                     = &PolicyDataSource{}
	_ datasource.DataSourceWithConfigValidators = &PolicyDataSource{}
)

func NewPolicyDataSource(retryWindow time.Duration) datasource.DataSource {
	if retryWindow <= 0 {
		retryWindow = notFoundRetryWindow
	}
	return &PolicyDataSource{notFoundRetryWindow: retryWindow}
}

// PolicyDataSource defines the data source implementation.
type PolicyDataSource struct {
	client              *client.ClientWithResponses
	notFoundRetryWindow time.Duration
}

func (d *PolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (d *PolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Keycard policy. A policy is a container for Cedar authorization rules within a zone.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy. Either `id` or `name` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this policy belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the policy. Either `id` or `name` must be provided, but not both.",
				Optional:            true,
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
			"latest_version_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the policy's latest version. May be empty when the policy has no published versions.",
				Computed:            true,
			},
		},
	}
}

func (d *PolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PolicyDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
	}
}

func (d *PolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicyModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var policy *client.Policy

	if !data.ID.IsNull() {
		// svc-pdp is eventually consistent, so a policy just created (or its
		// zone still provisioning) can 404 on a read replica briefly. A
		// persistent 404 is a genuine miss, so bound retries to a short window.
		getResp, err := callWithRetry(ctx, func() (*client.GetPolicyResponse, error) {
			return d.client.GetPolicyWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
		}, retryOnNotFound, withRetryWindow(d.notFoundRetryWindow))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
			return
		}

		if getResp.StatusCode() == 404 {
			resp.Diagnostics.AddError(
				"Policy Not Found",
				fmt.Sprintf("Policy with ID %s not found in zone %s", data.ID.ValueString(), data.ZoneID.ValueString()),
			)
			return
		}

		if getResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read policy, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
			)
			return
		}

		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to read policy, no response body")
			return
		}

		policy = getResp.JSON200
	} else {
		name := data.Name.ValueString()
		matches, err := findPoliciesByName(ctx, d.client, data.ZoneID.ValueString(), name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policies: %s", err))
			return
		}

		if len(matches) == 0 {
			resp.Diagnostics.AddError(
				"Policy Not Found",
				fmt.Sprintf("No policy found with name '%s' in zone '%s'", name, data.ZoneID.ValueString()),
			)
			return
		}

		if len(matches) > 1 {
			resp.Diagnostics.AddError(
				"Multiple Policies Found",
				fmt.Sprintf("Expected exactly 1 policy with name '%s' in zone '%s', but found %d. This indicates a data integrity issue.",
					name, data.ZoneID.ValueString(), len(matches)),
			)
			return
		}

		policy = &matches[0]
	}

	updatePolicyModelFromAPIResponse(policy, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// findPoliciesByName returns the policies in the zone whose name equals name
// exactly. query[name] is a case-insensitive substring match and the list is
// cursor-paginated, so it walks every page and filters client-side: an exact
// match can land beyond page 1 when other policy names contain the requested
// name as a substring. Stops early once two matches are found, since callers
// treat that as an error.
func findPoliciesByName(ctx context.Context, c *client.ClientWithResponses, zoneID, name string) ([]client.Policy, error) {
	var matches []client.Policy
	var after *string
	for {
		// A zone is provisioned asynchronously, so the list endpoint may 404
		// ("zone not found") briefly after the zone is created.
		listResp, err := callWithRetry(ctx, func() (*client.ListPoliciesResponse, error) {
			return c.ListPoliciesWithResponse(ctx, zoneID, &client.ListPoliciesParams{
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

		for _, p := range listResp.JSON200.Items {
			if p.Name == name {
				matches = append(matches, p)
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
