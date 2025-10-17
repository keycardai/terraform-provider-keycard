package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZoneDataSource{}

func NewZoneDataSource() datasource.DataSource {
	return &ZoneDataSource{}
}

// ZoneDataSource defines the data source implementation.
type ZoneDataSource struct {
	client *client.ClientWithResponses
}

func (d *ZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (d *ZoneDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about an existing Keycard zone by ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the zone.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the zone.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the zone's purpose.",
				Computed:            true,
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth2 configuration for the zone.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"pkce_required": schema.BoolAttribute{
						MarkdownDescription: "Whether PKCE (Proof Key for Code Exchange) is required for authorization code flows.",
						Computed:            true,
					},
					"dcr_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether Dynamic Client Registration (DCR) is enabled.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *ZoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the zone
	getResp, err := d.client.GetZoneWithResponse(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read zone, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(
			"Zone Not Found",
			fmt.Sprintf("Unable to find zone with ID %s. The zone may have been deleted or does not exist.", data.ID.ValueString()),
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

	// Update the model with the response data
	zone := getResp.JSON200
	data.ID = types.StringValue(zone.Id)
	data.Name = types.StringValue(zone.Name)
	data.Description = types.StringPointerValue(zone.Description)

	oauth2Data := OAuth2Model{
		PkceRequired: types.BoolPointerValue(zone.Oauth2PkceRequired),
		DcrEnabled:   types.BoolPointerValue(zone.Oauth2DcrEnabled),
	}

	oauth2Obj, diags := types.ObjectValueFrom(ctx, oauth2Data.AttributeTypes(), oauth2Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.OAuth2 = oauth2Obj

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
