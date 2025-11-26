package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ApplicationDataSource{}
var _ datasource.DataSourceWithConfigValidators = &ApplicationDataSource{}

func NewApplicationDataSource() datasource.DataSource {
	return &ApplicationDataSource{}
}

// ApplicationDataSource defines the data source implementation.
type ApplicationDataSource struct {
	client *client.ClientWithResponses
}

func (d *ApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *ApplicationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Keycard application. An application is a software system with an associated identity that can access resources.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the application. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this application belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the application.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the application's purpose. May be empty.",
				Computed:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "User-specified identifier for the application, typically its URL or URN. Must be unique within the zone. Either `id` or `identifier` must be provided, but not both.",
				Optional:            true,
				Computed:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata associated with the application. May be empty.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"docs_url": schema.StringAttribute{
						MarkdownDescription: "URL to documentation relevant to this application. May be empty.",
						Computed:            true,
					},
				},
			},
			"oauth2": schema.SingleNestedAttribute{
				MarkdownDescription: "OAuth2 configuration for the application. May be empty.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_uris": schema.ListAttribute{
						MarkdownDescription: "OAuth 2.0 redirect URIs for authorization code/token delivery. May be empty.",
						ElementType:         types.StringType,
						Computed:            true,
					},
				},
			},
			"traits": schema.ListAttribute{
				MarkdownDescription: "Traits of the application. Traits ascribe behaviors and characteristics to an application. May be empty.",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *ApplicationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ApplicationDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("identifier"),
		),
	}
}

func (d *ApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var application *client.Application

	if !data.ID.IsNull() {
		// Lookup by ID
		getResp, err := d.client.GetApplicationWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application, got error: %s", err))
			return
		}

		if getResp.StatusCode() == 404 {
			resp.Diagnostics.AddError(
				"Application Not Found",
				fmt.Sprintf("Application with ID %s not found in zone %s", data.ID.ValueString(), data.ZoneID.ValueString()),
			)
			return
		}

		if getResp.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read application, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
			)
			return
		}

		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Unable to read application, no response body")
			return
		}

		application = getResp.JSON200
	} else {
		// Lookup by identifier
		identifier := data.Identifier.ValueString()
		listResp, err := d.client.ListApplicationsWithResponse(ctx, data.ZoneID.ValueString(), &client.ListApplicationsParams{
			Identifier: &identifier,
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list applications: %s", err))
			return
		}

		if listResp.JSON200 == nil {
			resp.Diagnostics.AddError("API Error", "Received empty response from API")
			return
		}

		resultCount := len(listResp.JSON200.Items)
		if resultCount == 0 {
			resp.Diagnostics.AddError(
				"Application Not Found",
				fmt.Sprintf("No application found with identifier '%s' in zone '%s'", identifier, data.ZoneID.ValueString()),
			)
			return
		}

		if resultCount > 1 {
			resp.Diagnostics.AddError(
				"Multiple Applications Found",
				fmt.Sprintf("Expected exactly 1 application with identifier '%s' in zone '%s', but found %d. This indicates a data integrity issue.",
					identifier, data.ZoneID.ValueString(), resultCount),
			)
			return
		}

		application = &listResp.JSON200.Items[0]
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationModelFromAPIResponse(ctx, application, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
