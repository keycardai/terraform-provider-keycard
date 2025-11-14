package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

var _ datasource.DataSource = &KeyPolicyDataSource{}

func NewKeyPolicyDataSource() datasource.DataSource {
	return &KeyPolicyDataSource{}
}

type KeyPolicyDataSource struct {
	client *client.KeycardClient
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
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(12),
					stringvalidator.LengthAtMost(12), // AWS account IDs are always 12 digits
				},
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

	apiClient, ok := req.ProviderData.(*client.KeycardClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.KeycardClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
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

	orgId, err := d.client.GetOrganizationID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get organization ID: %s", err))
		return
	}

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
							principal["AWS"] = replaceAccountIDPlaceholder(v, accountID)
						case []interface{}:
							// Multiple AWS principal ARNs
							for i, arn := range v {
								if arnStr, ok := arn.(string); ok {
									v[i] = replaceAccountIDPlaceholder(arnStr, accountID)
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

// replaceAccountIDPlaceholder replaces the <YOUR_AWS_ACCOUNT_ID> placeholder with the actual AWS account ID
func replaceAccountIDPlaceholder(arn, accountID string) string {
	return strings.ReplaceAll(arn, "<YOUR_AWS_ACCOUNT_ID>", accountID)
}
