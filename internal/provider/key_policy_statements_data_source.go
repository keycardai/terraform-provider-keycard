package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

var _ datasource.DataSource = &KeyPolicyStatementsDataSource{}

func NewKeyPolicyStatementsDataSource() datasource.DataSource {
	return &KeyPolicyStatementsDataSource{}
}

type KeyPolicyStatementsDataSource struct {
	client *client.ClientWithResponses
}

type KeyPolicyStatementsDataSourceModel struct {
	PolicyStatements []types.String `tfsdk:"policy_statements"`
}

func (d *KeyPolicyStatementsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key_policy_statements"
}

func (d *KeyPolicyStatementsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches KMS key policy statements for customer-managed KMS encryption keys.",
		Attributes: map[string]schema.Attribute{
			"policy_statements": schema.ListAttribute{
				MarkdownDescription: "List of JSON-encoded key policy statements that can be used in AWS KMS key policies.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *KeyPolicyStatementsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KeyPolicyStatementsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KeyPolicyStatementsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the list or Organizations scoped to the user's credentials
	listOrgsParams := client.ListOrganizationsParams{}

	orgResp, err := d.client.ListOrganizationsWithResponse(ctx, &listOrgsParams)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to list organizations: %s", err))
		return
	}

	if orgResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(
			"No Organizations found",
			fmt.Sprintf("Unable to find any Organizations"),
		)
		return
	}

	if orgResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to list organizations: %s", err),
		)
		return
	}

	if orgResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to list organizations: no response body")
		return
	}

	// Use the first Organization ID from the list in the response
	// There should only be one since there is a 1:1 relationship between service accounts and orgs
	orgId := *orgResp.JSON200.Items[0].Id

	kpsResp, err := d.client.GetOrganizationKMSKeyPolicyStatementsWithResponse(ctx, orgId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to list KMS Key Policy Statements: %s", err))
		return
	}

	if kpsResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to list KMS Key Policy Statements: %s", err),
		)
		return
	}

	if kpsResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"API Error",
			"Unable to list KMS Key Policy Statements: no response body",
		)
		return
	}

	// Map API response to Terraform model
	policyStatements := make([]types.String, len(kpsResp.JSON200.PolicyStatements))
	for i, statement := range kpsResp.JSON200.PolicyStatements {
		policyStatements[i] = types.StringValue(statement)
	}
	data.PolicyStatements = policyStatements

	// Save data to the Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
