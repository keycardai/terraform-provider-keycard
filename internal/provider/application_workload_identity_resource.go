package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ApplicationWorkloadIdentityResource{}
var _ resource.ResourceWithImportState = &ApplicationWorkloadIdentityResource{}

func NewApplicationWorkloadIdentityResource() resource.Resource {
	return &ApplicationWorkloadIdentityResource{}
}

// ApplicationWorkloadIdentityResource defines the resource implementation.
type ApplicationWorkloadIdentityResource struct {
	client *client.ClientWithResponses
}

// ApplicationWorkloadIdentityModel describes the application workload identity data model.
type ApplicationWorkloadIdentityModel struct {
	ID            types.String `tfsdk:"id"`
	ZoneID        types.String `tfsdk:"zone_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	ProviderID    types.String `tfsdk:"provider_id"`
	Subject       types.String `tfsdk:"subject"`
}

func (r *ApplicationWorkloadIdentityResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_workload_identity"
}

func (r *ApplicationWorkloadIdentityResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a workload identity credential for a Keycard application. " +
			"This allows applications to authenticate using tokens from external identity providers " +
			"(Kubernetes, GitHub Actions, AWS EKS, etc.) with optional subject claim validation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the workload identity credential.",
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
			"provider_id": schema.StringAttribute{
				MarkdownDescription: "The provider that validates tokens for this credential. Changing this will replace the credential.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "The subject claim (sub) that must match in the bearer token. " +
					"Format depends on the token issuer:\n" +
					"  - Kubernetes: `system:serviceaccount:<namespace>:<service-account-name>`\n" +
					"  - GitHub Actions: `repo:<org>/<repo>:ref:refs/heads/<branch>`\n" +
					"  - AWS EKS: `system:serviceaccount:<namespace>:<service-account-name>`\n\n",
				Required: true,
			},
		},
	}
}

func (r *ApplicationWorkloadIdentityResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ApplicationWorkloadIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationWorkloadIdentityModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the create request for a token-type credential
	subjectValue := data.Subject.ValueString()
	tokenCreate := client.ApplicationCredentialCreateToken{
		ApplicationId: data.ApplicationID.ValueString(),
		Type:          client.ApplicationCredentialCreateTokenTypeToken,
		ProviderId:    data.ProviderID.ValueString(),
		Subject:       &subjectValue,
	}

	createReq := client.ApplicationCredentialCreate{}
	err := createReq.FromApplicationCredentialCreateToken(tokenCreate)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to construct application workload identity request body, got error: %s", err))
		return
	}

	// Create the credential
	createResp, err := r.client.CreateApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create application workload identity, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create application workload identity, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create application workload identity, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationWorkloadIdentityModelFromCreateResponse(createResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationWorkloadIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationWorkloadIdentityModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the credential
	getResp, err := r.client.GetApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application workload identity, got error: %s", err))
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
			fmt.Sprintf("Unable to read application workload identity, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read application workload identity, no response body")
		return
	}

	// Update the model with the response data using the same helper as Create
	resp.Diagnostics.Append(updateApplicationWorkloadIdentityModelFromAPIResponse(getResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationWorkloadIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ApplicationWorkloadIdentityModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the update request for token credential
	tokenUpdateType := client.Token
	tokenUpdate := client.TokenCredentialUpdate{
		Type:    &tokenUpdateType,
		Subject: StringValueNullable(data.Subject),
	}

	updateReq := client.ApplicationCredentialUpdate{}
	err := updateReq.FromTokenCredentialUpdate(tokenUpdate)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to construct application workload identity update request body, got error: %s", err))
		return
	}

	// Update the credential
	updateResp, err := r.client.UpdateApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update application workload identity, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update application workload identity, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update application workload identity, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationWorkloadIdentityModelFromAPIResponse(updateResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationWorkloadIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationWorkloadIdentityModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the credential
	deleteResp, err := r.client.DeleteApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application workload identity, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete application workload identity, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *ApplicationWorkloadIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/application-credentials/{credential-id}
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 || parts[0] != "zones" || parts[2] != "application-credentials" || parts[1] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zones/{zone-id}/application-credentials/{credential-id}', got: %s", req.ID),
		)
		return
	}

	zoneID := parts[1]
	credentialID := parts[3]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), credentialID)...)
}
