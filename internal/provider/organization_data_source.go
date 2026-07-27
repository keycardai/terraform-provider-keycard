package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

var _ datasource.DataSource = &OrganizationDataSource{}

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

type OrganizationDataSource struct {
	client *client.ClientWithResponses
}

type OrganizationDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Label      types.String `tfsdk:"label"`
	SSOEnabled types.Bool   `tfsdk:"sso_enabled"`
	ZoneID     types.String `tfsdk:"zone_id"`
}

func (d *OrganizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns the Keycard organization that the authenticated identity belongs to.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the organization.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name for the organization.",
				Computed:            true,
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "A domain name segment for the organization.",
				Computed:            true,
			},
			"sso_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether SSO is enabled for the organization.",
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the builtin zone for the organization.",
				Computed:            true,
			},
		},
	}
}

func (d *OrganizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = apiClient
}

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgID, err := GetOrganizationID(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get organization ID: %s", err))
		return
	}

	getResp, err := d.client.GetOrganizationWithResponse(ctx, orgID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read organization, got error: %s", err))
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read organization, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"API Error",
			"Unable to read organization: no response body",
		)
		return
	}

	// Map API response to Terraform model
	org := getResp.JSON200
	data.ID = types.StringPointerValue(org.Id)
	data.Name = types.StringPointerValue(org.Name)
	data.Label = types.StringPointerValue(org.Label)
	data.SSOEnabled = types.BoolPointerValue(org.SsoEnabled)
	data.ZoneID = types.StringPointerValue(org.ZoneId)

	// Save data to the Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
