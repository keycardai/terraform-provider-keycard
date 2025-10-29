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
var _ datasource.DataSource = &ResourceDataSource{}
var _ datasource.DataSourceWithValidateConfig = &ResourceDataSource{}

func NewResourceDataSource() datasource.DataSource {
	return &ResourceDataSource{}
}

// ResourceDataSource defines the data source implementation.
type ResourceDataSource struct {
	client *client.ClientWithResponses
}

func (d *ResourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (d *ResourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Keycard resource. A resource is a system that exposes protected information or functionality requiring authentication.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the resource. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this resource belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the resource.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the resource's purpose. May be empty.",
				Computed:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "User-specified identifier for the resource, typically its URL or URN. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"credential_provider_id": schema.StringAttribute{
				MarkdownDescription: "The provider that issues credentials for accessing this resource. May be empty.",
				Computed:            true,
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The application that provides this resource. May be empty.",
				Computed:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata associated with the resource. May be empty.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"docs_url": schema.StringAttribute{
						MarkdownDescription: "URL to documentation relevant to this resource. May be empty.",
						Computed:            true,
					},
				},
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth2 configuration for the resource. May be empty.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"scopes": schema.ListAttribute{
						MarkdownDescription: "OAuth2 scopes required to access this resource. Must match scopes configured in the authorization server. May be empty.",
						ElementType:         types.StringType,
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *ResourceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ResourceDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var data ResourceModel

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

func (d *ResourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var resource *client.Resource

	if !data.ID.IsNull() {
		// Lookup by ID
		getResp, err := d.client.GetResourceWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read resource, got error: %s", err))
			return
		}

		if getResp.StatusCode() == 404 {
			resp.Diagnostics.AddError(
				"Resource Not Found",
				fmt.Sprintf("Resource with ID %s not found in zone %s", data.ID.ValueString(), data.ZoneID.ValueString()),
			)
			return
		}

		if getResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read resource, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
			)
			return
		}

		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to read resource, no response body")
			return
		}

		resource = getResp.JSON200
	} else {
		// Lookup by identifier
		identifier := data.Identifier.ValueString()
		listResp, err := d.client.ListResourcesWithResponse(ctx, data.ZoneID.ValueString(), &client.ListResourcesParams{
			Identifier: &identifier,
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list resources: %s", err))
			return
		}

		if listResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Received empty response from API")
			return
		}

		resultCount := len(listResp.JSON200.Items)
		if resultCount == 0 {
			resp.Diagnostics.AddError(
				"Resource Not Found",
				fmt.Sprintf("No resource found with identifier '%s' in zone '%s'", identifier, data.ZoneID.ValueString()),
			)
			return
		}

		if resultCount > 1 {
			resp.Diagnostics.AddError(
				"Multiple Resources Found",
				fmt.Sprintf("Expected exactly 1 resource with identifier '%s' in zone '%s', but found %d. This indicates a data integrity issue.",
					identifier, data.ZoneID.ValueString(), resultCount),
			)
			return
		}

		resource = &listResp.JSON200.Items[0]
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateResourceModelFromAPIResponse(ctx, resource, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
