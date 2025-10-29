package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ProviderDataSource{}
var _ datasource.DataSourceWithValidateConfig = &ProviderDataSource{}

func NewProviderDataSource() datasource.DataSource {
	return &ProviderDataSource{}
}

// ProviderDataSource defines the data source implementation.
type ProviderDataSource struct {
	client *client.ClientWithResponses
}

// ProviderDataSourceModel describes the data source data model.
// Note: This model excludes client_secret since it's write-only and not returned by the API.
type ProviderDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ZoneID      types.String `tfsdk:"zone_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Identifier  types.String `tfsdk:"identifier"`
	ClientID    types.String `tfsdk:"client_id"`
	OAuth2      types.Object `tfsdk:"oauth2"`
}

func (d *ProviderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provider"
}

func (d *ProviderDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Keycard provider. A provider is a system that supplies access to resources and allows actors (users or applications) to authenticate.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the provider. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this provider belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the provider.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the provider's purpose. May be empty.",
				Computed:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "User-specified identifier, unique within the zone. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth 2.0 client identifier. May be empty.",
				Computed:            true,
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth 2.0 protocol configuration. May be empty.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"authorization_endpoint": schema.StringAttribute{
						MarkdownDescription: "OAuth 2.0 Authorization endpoint URL. May be empty.",
						Computed:            true,
					},
					"token_endpoint": schema.StringAttribute{
						MarkdownDescription: "OAuth 2.0 Token endpoint URL. May be empty.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *ProviderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProviderDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var data ProviderDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate exactly one of id or identifier is provided
	if data.ID.IsNull() && data.Identifier.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Missing Required Attribute",
			"Either 'id' or 'identifier' must be provided",
		)
		return
	}

	if !data.ID.IsNull() && !data.Identifier.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting Attributes",
			"Cannot provide both 'id' and 'identifier'",
		)
		return
	}
}

func (d *ProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProviderDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var provider *client.Provider

	if !data.ID.IsNull() {
		// Lookup by ID
		getResp, err := d.client.GetProviderWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read provider, got error: %s", err))
			return
		}

		if getResp.StatusCode() == 404 {
			resp.Diagnostics.AddError(
				"Provider Not Found",
				fmt.Sprintf("Provider with ID %s not found in zone %s", data.ID.ValueString(), data.ZoneID.ValueString()),
			)
			return
		}

		if getResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read provider, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
			)
			return
		}

		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to read provider, no response body")
			return
		}

		provider = getResp.JSON200
	} else {
		// Lookup by identifier
		identifier := data.Identifier.ValueString()
		listResp, err := d.client.ListProvidersWithResponse(ctx, data.ZoneID.ValueString(), &client.ListProvidersParams{
			Identifier: &identifier,
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list providers: %s", err))
			return
		}

		if listResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Received empty response from API")
			return
		}

		resultCount := len(listResp.JSON200.Items)
		if resultCount == 0 {
			resp.Diagnostics.AddError(
				"Provider Not Found",
				fmt.Sprintf("No provider found with identifier '%s' in zone '%s'", identifier, data.ZoneID.ValueString()),
			)
			return
		}

		if resultCount > 1 {
			resp.Diagnostics.AddError(
				"Multiple Providers Found",
				fmt.Sprintf("Expected exactly 1 provider with identifier '%s' in zone '%s', but found %d. This indicates a data integrity issue.",
					identifier, data.ZoneID.ValueString(), resultCount),
			)
			return
		}

		provider = &listResp.JSON200.Items[0]
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateProviderDataSourceModelFromAPIResponse(ctx, provider, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
