package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ApplicationWorkloadIdentityDataSource{}

func NewApplicationWorkloadIdentityDataSource() datasource.DataSource {
	return &ApplicationWorkloadIdentityDataSource{}
}

// ApplicationWorkloadIdentityDataSource defines the data source implementation.
type ApplicationWorkloadIdentityDataSource struct {
	client *client.ClientWithResponses
}

func (d *ApplicationWorkloadIdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_workload_identity"
}

func (d *ApplicationWorkloadIdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a workload identity credential for a Keycard application. " +
			"This allows you to reference existing workload identity credentials that authenticate using " +
			"tokens from external identity providers (Kubernetes, GitHub Actions, AWS EKS, etc.).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the workload identity credential.",
				Required:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this credential belongs to.",
				Required:            true,
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The application this credential belongs to.",
				Computed:            true,
			},
			"provider_id": schema.StringAttribute{
				MarkdownDescription: "The provider that validates tokens for this credential.",
				Computed:            true,
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "The subject claim (sub) that must match in the bearer token. " +
					"Empty when any token from the provider is accepted. " +
					"Format depends on the token issuer:\n" +
					"  - Kubernetes: `system:serviceaccount:<namespace>:<service-account-name>`\n" +
					"  - GitHub Actions: `repo:<org>/<repo>:ref:refs/heads/<branch>`\n" +
					"  - AWS EKS: `system:serviceaccount:<namespace>:<service-account-name>`\n\n",
				Computed: true,
			},
		},
	}
}

func (d *ApplicationWorkloadIdentityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ApplicationWorkloadIdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationWorkloadIdentityModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the credential
	getResp, err := d.client.GetApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application workload identity, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(
			"Application Workload Identity Not Found",
			fmt.Sprintf("Application workload identity with ID %s not found in zone %s", data.ID.ValueString(), data.ZoneID.ValueString()),
		)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read application workload identity, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read application workload identity, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationWorkloadIdentityModelFromAPIResponse(getResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
