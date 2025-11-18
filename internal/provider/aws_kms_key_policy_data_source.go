package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

var _ datasource.DataSource = &AwsKmsKeyPolicyDataSource{}

func NewAwsKmsKeyPolicyDataSource() datasource.DataSource {
	return &AwsKmsKeyPolicyDataSource{}
}

type AwsKmsKeyPolicyDataSource struct {
	client *client.ClientWithResponses
}

type AwsKmsKeyPolicyDataSourceModel struct {
	AccountID types.String `tfsdk:"account_id"`
	Policy    types.String `tfsdk:"policy"`
}

func (d *AwsKmsKeyPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_kms_key_policy"
}

func (d *AwsKmsKeyPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns an AWS KMS key policy that grants Keycard permissions to Encrypt and Decrypt operations on the key scoped to this Keycard organization as well as DescribeKey permissions on the key.",
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				MarkdownDescription: "AWS account ID to allow admin access in the KMS key policy. ",
				Required:            true,
			},
			"policy": schema.StringAttribute{
				MarkdownDescription: "JSON-encoded AWS policy document that can be used with the AWS terraform provider.",
				Computed:            true,
			},
		},
	}
}

func (d *AwsKmsKeyPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AwsKmsKeyPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AwsKmsKeyPolicyDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	listOrgsParams := client.ListOrganizationsParams{}

	orgResp, err := d.client.ListOrganizationsWithResponse(ctx, &listOrgsParams)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to list organizations: %s", err))
		return
	}

	if orgResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to list organizations: %d", orgResp.StatusCode()),
		)
		return
	}

	if orgResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to list organizations: no response body")
		return
	}

	// Use the first Organization ID from the list in the response.
	// The access token is scoped to only 1 organization so there should only be one item in this list.
	if len(orgResp.JSON200.Items) != 1 {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unexpected number of organizations in list response: %v", len(orgResp.JSON200.Items)))
		return
	}

	thisOrg := orgResp.JSON200.Items[0]
	if thisOrg.Id == nil {
		resp.Diagnostics.AddError("Client Error", "Missing organization ID")
		return
	}

	kpResp, err := d.client.GetOrganizationKMSKeyPolicyWithResponse(ctx, *thisOrg.Id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to get KMS Key Policy: %s", err))
		return
	}

	if kpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to get KMS Key Policy, response code: %d", kpResp.StatusCode()),
		)
		return
	}

	if kpResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"API Error",
			"Unable to get KMS Key Policy: no response body",
		)
		return
	}

	// Replace the account ID placeholder in the policy with the one provided
	accountID := data.AccountID.ValueString()

	policyDocument := strings.ReplaceAll(kpResp.JSON200.Policy, "<YOUR_AWS_ACCOUNT_ID>", accountID)

	// Validate the string replace still returns valid JSON
	if !json.Valid([]byte(policyDocument)) {
		resp.Diagnostics.AddError("Client Error", "Policy document returned invalid JSON")
		return
	}

	// Map API response to Terraform model
	data.Policy = types.StringValue(policyDocument)

	// Save data to the Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
