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

var _ datasource.DataSource = &KeyPolicyDataSource{}

func NewKeyPolicyDataSource() datasource.DataSource {
	return &KeyPolicyDataSource{}
}

type KeyPolicyDataSource struct {
	client *client.ClientWithResponses
}

type KeyPolicyDataSourceModel struct {
	AccountID types.String `tfsdk:"account_id"`
	Policy    types.String `tfsdk:"policy"`
}

func (d *KeyPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key_policy"
}

func (d *KeyPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a KMS key policy for customer-managed KMS encryption keys.",
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				MarkdownDescription: "AWS account ID to allow admin access in the KMS key policy",
				Required:            true,
			},
			"policy": schema.StringAttribute{
				MarkdownDescription: "JSON-encoded key policy that can be used in AWS KMS key policies.",
				Computed:            true,
			},
		},
	}
}

func (d *KeyPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KeyPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KeyPolicyDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// // Get the list or Organizations scoped to the user's credentials
	// listOrgsParams := client.ListOrganizationsParams{}

	// orgResp, err := d.client.ListOrganizationsWithResponse(ctx, &listOrgsParams)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to list organizations: %s", err))
	// 	return
	// }

	// if orgResp.StatusCode() == 404 {
	// 	resp.Diagnostics.AddError(
	// 		"No Organizations found",
	// 		fmt.Sprintf("Unable to find any Organizations"),
	// 	)
	// 	return
	// }

	// if orgResp.StatusCode() != 200 {
	// 	resp.Diagnostics.AddError(
	// 		"API Error",
	// 		fmt.Sprintf("Failed to list organizations: %d", orgResp.StatusCode()),
	// 	)
	// 	return
	// }

	// if orgResp.JSON200 == nil {
	// 	resp.Diagnostics.AddError("API Error", "Unable to list organizations: no response body")
	// 	return
	// }

	// // Use the first Organization ID from the list in the response
	// // There should only be one since there is a 1:1 relationship between service accounts and orgs
	// orgId := *orgResp.JSON200.Items[0].Id
	orgId := "nicksimkos-organization-pr1xp"

	kpResp, err := d.client.GetOrganizationKMSKeyPolicyWithResponse(ctx, orgId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to list KMS Key Policy Statements: %s", err))
		return
	}

	if kpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to list KMS Key Policy Statements; response code: %d", kpResp.StatusCode()),
		)
		return
	}

	if kpResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"API Error",
			"Unable to list KMS Key Policy Statements: no response body",
		)
		return
	}

	// Replace the account ID placeholder in the policy with the one provided
	accountID := data.AccountID.ValueString()
	var policyJSON map[string]interface{}

	// Unmarshal the policy JSON string into a map
	err = json.Unmarshal([]byte(kpResp.JSON200.Policy), &policyJSON)
	if err != nil {
		resp.Diagnostics.AddError("JSON error", fmt.Sprintf("Failed to unmarshal policy JSON: %s", err))
		return
	}

	// Replace <YOUR_AWS_ACCOUNT_ID> with the actual account ID in the policy
	if statements, ok := policyJSON["Statement"].([]interface{}); ok {
		for _, stmt := range statements {
			if statement, ok := stmt.(map[string]interface{}); ok {
				if principal, ok := statement["Principal"].(map[string]interface{}); ok {
					// Check AWS principal (can be string or array)
					if awsPrincipal, ok := principal["AWS"]; ok {
						switch v := awsPrincipal.(type) {
						case string:
							// Single AWS principal ARN
							principal["AWS"] = strings.ReplaceAll(v, "<YOUR_AWS_ACCOUNT_ID>", accountID)
						case []interface{}:
							// Multiple AWS principal ARNs
							for i, arn := range v {
								if arnStr, ok := arn.(string); ok {
									v[i] = strings.ReplaceAll(arnStr, "<YOUR_AWS_ACCOUNT_ID>", accountID)
								}
							}
						}
					}
				}
			}
		}
	}

	// Marshal the modified policy back to JSON
	modifiedPolicyBytes, err := json.Marshal(policyJSON)
	if err != nil {
		resp.Diagnostics.AddError("JSON error", fmt.Sprintf("Failed to marshal modified policy JSON: %s", err))
		return
	}

	// Map API response to Terraform model
	data.Policy = types.StringValue(string(modifiedPolicyBytes))

	// Save data to the Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
