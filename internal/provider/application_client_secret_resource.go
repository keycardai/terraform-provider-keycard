package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ApplicationClientSecretResource{}

func NewApplicationClientSecretResource() resource.Resource {
	return &ApplicationClientSecretResource{}
}

// ApplicationClientSecretResource defines the resource implementation.
type ApplicationClientSecretResource struct {
	client *client.KeycardClient
}

// ApplicationClientSecretModel describes the application client secret data model.
type ApplicationClientSecretModel struct {
	ID            types.String `tfsdk:"id"`
	ZoneID        types.String `tfsdk:"zone_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	ClientID      types.String `tfsdk:"client_id"`
	ClientSecret  types.String `tfsdk:"client_secret"`
}

func (r *ApplicationClientSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_client_secret"
}

func (r *ApplicationClientSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages OAuth 2.0 client credentials (client_id/client_secret pair) for a Keycard application. " +
			"These credentials can be used for OAuth 2.0 client credentials flow. " +
			"**Important**: The client_secret is only available immediately after creation. " +
			"If Terraform state is lost, the resource must be recreated as the secret cannot be retrieved. " +
			"~> **Note** Import is not supported for this resource.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the credential.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this credential belongs to. Changing this will replace the credential.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The application this credential belongs to. Changing this will replace the credential.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The OAuth 2.0 client ID. This value is auto-generated and can be used as the username for client credentials flow.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The OAuth 2.0 client secret. This value is only returned on creation and cannot be retrieved later. Store it securely.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ApplicationClientSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.KeycardClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.KeycardClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ApplicationClientSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationClientSecretModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the create request for a password-type credential
	passwordCreate := client.ApplicationCredentialCreatePassword{
		ApplicationId: data.ApplicationID.ValueString(),
		Type:          client.ApplicationCredentialCreatePasswordTypePassword,
	}

	createReq := client.ApplicationCredentialCreate{}
	err := createReq.FromApplicationCredentialCreatePassword(passwordCreate)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to construct application client secret request body, got error: %s", err))
		return
	}

	// Create the credential
	createResp, err := r.client.CreateApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create application client secret, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create application client secret, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create application client secret, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationClientSecretModelFromCreateResponse(createResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationClientSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationClientSecretModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the credential
	getResp, err := r.client.GetApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application client secret, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Credential was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read application client secret, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read application client secret, no response body")
		return
	}

	// Update the model with the response data
	// Note: client_secret is preserved from state as the API doesn't return the password
	resp.Diagnostics.Append(updateApplicationClientSecretModelFromAPIResponse(getResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationClientSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource does not support updates - all attributes require replacement
	// This method should never be called due to RequiresReplace plan modifiers
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Application client secrets are immutable. Any changes require resource replacement.",
	)
}

func (r *ApplicationClientSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationClientSecretModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the credential
	deleteResp, err := r.client.DeleteApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application client secret, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete application client secret, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}
