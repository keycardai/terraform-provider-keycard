package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ApplicationPublicCredentialResource{}
	_ resource.ResourceWithImportState = &ApplicationPublicCredentialResource{}
)

func NewApplicationPublicCredentialResource() resource.Resource {
	return &ApplicationPublicCredentialResource{}
}

// ApplicationPublicCredentialResource defines the resource implementation.
type ApplicationPublicCredentialResource struct {
	client *client.ClientWithResponses
}

// ApplicationPublicCredentialModel describes the application public credential data model.
type ApplicationPublicCredentialModel struct {
	ID            types.String `tfsdk:"id"`
	ZoneID        types.String `tfsdk:"zone_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Identifier    types.String `tfsdk:"identifier"`
}

func (r *ApplicationPublicCredentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_public_credential"
}

func (r *ApplicationPublicCredentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages public application credentials for a Keycard application. " +
			"Public credentials have no secret — they consist only of a client identifier. " +
			"This credential type is suitable for public OAuth clients.",

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
			"identifier": schema.StringAttribute{
				MarkdownDescription: "The client identifier for this public credential, " +
					"used as the OAuth 2.0 client_id in authorization requests. " +
					"Changing this will replace the credential.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *ApplicationPublicCredentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationPublicCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationPublicCredentialModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the create request for a public-type credential
	publicCreate := client.ApplicationCredentialCreatePublic{
		ApplicationId: data.ApplicationID.ValueString(),
		Identifier:    data.Identifier.ValueStringPointer(), // *string per generated client; never nil since field is Required
		Type:          client.ApplicationCredentialCreatePublicTypePublic,
	}

	createReq := client.ApplicationCredentialCreate{}
	err := createReq.FromApplicationCredentialCreatePublic(publicCreate)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to construct application public credential request body, got error: %s", err))
		return
	}

	// Create the credential
	createResp, err := r.client.CreateApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create application public credential, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create application public credential, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to create application public credential, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationPublicCredentialModelFromCreateResponse(createResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationPublicCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationPublicCredentialModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the credential
	getResp, err := r.client.GetApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application public credential, got error: %s", err))
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
			fmt.Sprintf("Unable to read application public credential, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read application public credential, no response body")
		return
	}

	// Update the model with the response data
	resp.Diagnostics.Append(updateApplicationPublicCredentialModelFromAPIResponse(getResp.JSON200, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationPublicCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource does not support updates - all attributes require replacement
	// This method should never be called due to RequiresReplace plan modifiers
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Application public credentials are immutable. Any changes require resource replacement.",
	)
}

func (r *ApplicationPublicCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationPublicCredentialModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the credential
	deleteResp, err := r.client.DeleteApplicationCredentialWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application public credential, got error: %s", err))
		return
	}

	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete application public credential, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *ApplicationPublicCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/application-credentials/{credential-id}.
	// Note: application_id is not encoded in the import string. After ImportState sets
	// zone_id and id, the framework calls Read, which fetches the credential from the API
	// and populates application_id from the response.
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
