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
var _ datasource.DataSource = &PolicySchemaDataSource{}

func NewPolicySchemaDataSource(retryWindow time.Duration) datasource.DataSource {
	if retryWindow <= 0 {
		retryWindow = notFoundRetryWindow
	}
	return &PolicySchemaDataSource{notFoundRetryWindow: retryWindow}
}

// PolicySchemaDataSource defines the data source implementation.
type PolicySchemaDataSource struct {
	client              *client.ClientWithResponses
	notFoundRetryWindow time.Duration
}

// PolicySchemaDataSourceModel describes the data source data model.
type PolicySchemaDataSourceModel struct {
	ZoneID      types.String `tfsdk:"zone_id"`
	Version     types.String `tfsdk:"version"`
	Status      types.String `tfsdk:"status"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
	CedarSchema types.String `tfsdk:"cedar_schema"`
}

func (d *PolicySchemaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_schema"
}

func (d *PolicySchemaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Cedar policy schema for a zone. Policy schemas define the entity model " +
			"(users, applications, resources) that Cedar policies are validated against. Schemas are managed " +
			"by the Keycard platform; use this data source to look up the zone's default schema version, or " +
			"a specific version, when creating policy versions.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the zone.",
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Schema version to fetch (e.g. `2026-02-24`). When omitted, the zone's default schema is returned.",
				Optional:            true,
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Lifecycle status of the schema version: `active`, `deprecated`, or `archived`. " +
					"New policy versions cannot be created against archived schemas.",
				Computed: true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this schema version is the zone's default.",
				Computed:            true,
			},
			"cedar_schema": schema.StringAttribute{
				MarkdownDescription: "The Cedar schema in human-readable Cedar syntax.",
				Computed:            true,
			},
		},
	}
}

func (d *PolicySchemaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PolicySchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicySchemaDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var apiSchema *client.SchemaVersionWithZoneInfo

	if !data.Version.IsNull() {
		format := client.GetPolicySchemaParamsFormatCedar
		// svc-pdp is eventually consistent: a 404 may mean the zone is still
		// provisioning, or that the requested version row hasn't propagated to
		// the replica serving this read yet. Retry transient 404s over a short
		// window that covers propagation lag; here a 404 is most often a real
		// miss, so surface it quickly instead of waiting out the full window.
		getResp, err := callWithRetry(ctx, func() (*client.GetPolicySchemaResponse, error) {
			return d.client.GetPolicySchemaWithResponse(ctx, data.ZoneID.ValueString(), data.Version.ValueString(), &client.GetPolicySchemaParams{
				Format: &format,
			})
		}, retryOnNotFound, withRetryWindow(d.notFoundRetryWindow))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy schema, got error: %s", err))
			return
		}

		if getResp.StatusCode() == 404 {
			resp.Diagnostics.AddError(
				"Policy Schema Not Found",
				fmt.Sprintf("Unable to find policy schema version %s in zone %s.", data.Version.ValueString(), data.ZoneID.ValueString()),
			)
			return
		}

		if getResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read policy schema, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
			)
			return
		}

		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to read policy schema, no response body")
			return
		}

		apiSchema = getResp.JSON200
	} else {
		format := client.ListPolicySchemasParamsFormatCedar
		isDefault := true
		// A 404 here can only mean the zone hasn't finished provisioning.
		listResp, err := callWithRetry(ctx, func() (*client.ListPolicySchemasResponse, error) {
			return d.client.ListPolicySchemasWithResponse(ctx, data.ZoneID.ValueString(), &client.ListPolicySchemasParams{
				Format:    &format,
				IsDefault: &isDefault,
			})
		}, retryOnNotFound)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policy schemas, got error: %s", err))
			return
		}

		if listResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to list policy schemas, got status %d: %s", listResp.StatusCode(), string(listResp.Body)),
			)
			return
		}

		if listResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to list policy schemas, no response body")
			return
		}

		if len(listResp.JSON200.Items) == 0 {
			resp.Diagnostics.AddError(
				"Default Policy Schema Not Found",
				fmt.Sprintf("Zone %s has no default policy schema. Specify a version explicitly.", data.ZoneID.ValueString()),
			)
			return
		}

		if len(listResp.JSON200.Items) > 1 {
			resp.Diagnostics.AddError(
				"Multiple Default Policy Schemas Found",
				fmt.Sprintf("Zone %s returned %d default policy schemas, expected exactly one. Please report this issue to the provider developers.", data.ZoneID.ValueString(), len(listResp.JSON200.Items)),
			)
			return
		}

		apiSchema = &listResp.JSON200.Items[0]
	}

	data.Version = types.StringValue(apiSchema.Version)
	data.Status = types.StringValue(string(apiSchema.Status))
	data.IsDefault = types.BoolValue(apiSchema.IsDefault)
	data.CedarSchema = NullableStringValue(apiSchema.CedarSchema)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
