package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZoneUserIdentityConfigDataSource{}

func NewZoneUserIdentityConfigDataSource() datasource.DataSource {
	return &ZoneUserIdentityConfigDataSource{}
}

// ZoneUserIdentityConfigDataSource defines the data source implementation.
type ZoneUserIdentityConfigDataSource struct {
	client *client.KeycardClient
}

// ZoneUserIdentityConfigDataSourceModel describes the data source data model.
type ZoneUserIdentityConfigDataSourceModel struct {
	ZoneID     types.String `tfsdk:"zone_id"`
	ProviderID types.String `tfsdk:"provider_id"`
}

func (d *ZoneUserIdentityConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_user_identity_config"
}

func (d *ZoneUserIdentityConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the user identity provider configuration for a Keycard zone.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the zone to read the configuration from.",
				Required:            true,
			},
			"provider_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the provider configured for user authentication in this zone.",
				Computed:            true,
			},
		},
	}
}

func (d *ZoneUserIdentityConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.KeycardClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.KeycardClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ZoneUserIdentityConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneUserIdentityConfigDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the zone
	getResp, err := d.client.GetZoneWithResponse(ctx, data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read zone, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(
			"Zone Not Found",
			fmt.Sprintf("Zone with ID %s was not found", data.ZoneID.ValueString()),
		)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read zone, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read zone, no response body")
		return
	}

	zone := getResp.JSON200

	// Check if the zone has a user identity provider configured
	if zone.UserIdentityProviderId == nil || *zone.UserIdentityProviderId == "" {
		resp.Diagnostics.AddError(
			"No Provider Configured",
			fmt.Sprintf("Zone %s does not have a user identity provider configured", data.ZoneID.ValueString()),
		)
		return
	}

	// Set provider_id from the zone's current configuration
	data.ProviderID = types.StringPointerValue(zone.UserIdentityProviderId)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
