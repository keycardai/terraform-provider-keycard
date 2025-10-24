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
var _ datasource.DataSource = &ZoneDirectoryDataSource{}

func NewZoneDirectoryDataSource() datasource.DataSource {
	return &ZoneDirectoryDataSource{}
}

// ZoneDirectoryDataSource defines the data source implementation.
type ZoneDirectoryDataSource struct {
	client *client.ClientWithResponses
}

// ZoneDirectoryDataSourceModel describes the data source data model.
type ZoneDirectoryDataSourceModel struct {
	ZoneID     types.String `tfsdk:"zone_id"`
	ProviderID types.String `tfsdk:"provider_id"`
}

func (d *ZoneDirectoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_directory"
}

func (d *ZoneDirectoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches the built-in Keycard Directory provider for a zone. Every zone has a built-in directory provider for managing user identities.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone ID to fetch the directory provider for.",
				Required:            true,
			},
			"provider_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the built-in directory provider for this zone.",
				Computed:            true,
			},
		},
	}
}

func (d *ZoneDirectoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZoneDirectoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneDirectoryDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// List providers filtered by type=keycard-directory
	providerType := client.ListProvidersParamsTypeKeycardDirectory
	params := &client.ListProvidersParams{
		Type: &providerType,
	}

	listResp, err := d.client.ListProvidersWithResponse(ctx, data.ZoneID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list providers, got error: %s", err))
		return
	}

	if listResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(
			"Zone Not Found",
			fmt.Sprintf("Zone with ID %s not found", data.ZoneID.ValueString()),
		)
		return
	}

	if listResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to list providers, got status %d: %s", listResp.StatusCode(), string(listResp.Body)),
		)
		return
	}

	if listResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to list providers, no response body")
		return
	}

	// Check if we got any providers
	if len(listResp.JSON200.Items) == 0 {
		resp.Diagnostics.AddError(
			"Directory Provider Not Found",
			fmt.Sprintf("Zone %s does not have a directory provider. This should not happen as every zone should have a built-in directory provider.", data.ZoneID.ValueString()),
		)
		return
	}

	// Get the first (and should be only) directory provider
	directoryProvider := listResp.JSON200.Items[0]
	data.ProviderID = types.StringValue(directoryProvider.Id)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
