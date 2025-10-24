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
var _ resource.Resource = &ResourceDependencyResource{}
var _ resource.ResourceWithImportState = &ResourceDependencyResource{}

func NewResourceDependencyResource() resource.Resource {
	return &ResourceDependencyResource{}
}

// ResourceDependencyResource defines the resource implementation.
type ResourceDependencyResource struct {
	client *client.ClientWithResponses
}

// ResourceDependencyModel describes the resource data model.
type ResourceDependencyModel struct {
	ZoneID        types.String `tfsdk:"zone_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	ResourceID    types.String `tfsdk:"resource_id"`
}

func (r *ResourceDependencyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_dependency"
}

func (r *ResourceDependencyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a resource dependency for an application. A resource dependency allows an application to generate delegated user grants for accessing the resource.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this resource dependency belongs to. Changing this will replace the resource dependency.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The application that needs access to the resource. Changing this will replace the resource dependency.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "The resource that the application needs to access. Changing this will replace the resource dependency.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ResourceDependencyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceDependencyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceDependencyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Add the resource dependency
	createResp, err := r.client.AddApplicationDependencyWithResponse(
		ctx,
		data.ZoneID.ValueString(),
		data.ApplicationID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create resource dependency, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 204 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create resource dependency, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceDependencyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceDependencyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the resource dependency
	getResp, err := r.client.GetApplicationDependencyWithResponse(
		ctx,
		data.ZoneID.ValueString(),
		data.ApplicationID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read resource dependency, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Resource dependency was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read resource dependency, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read resource dependency, no response body")
		return
	}

	// The dependency exists - state is already correct, no need to update
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceDependencyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields are immutable (RequiresReplace), so this should never be called.
	// If it is called, we just need to read the plan and set it as the new state.
	var data ResourceDependencyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceDependencyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceDependencyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the resource dependency
	deleteResp, err := r.client.RemoveApplicationDependencyWithResponse(
		ctx,
		data.ZoneID.ValueString(),
		data.ApplicationID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete resource dependency, got error: %s", err))
		return
	}

	// Accept both 204 (deleted) and 404 (already gone) as success
	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete resource dependency, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *ResourceDependencyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/applications/{application-id}/dependencies/{resource-id}
	parts := strings.Split(req.ID, "/")
	if len(parts) != 6 || parts[0] != "zones" || parts[2] != "applications" || parts[4] != "dependencies" || parts[1] == "" || parts[3] == "" || parts[5] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zones/{zone-id}/applications/{application-id}/dependencies/{resource-id}', got: %s", req.ID),
		)
		return
	}

	zoneID := parts[1]
	applicationID := parts[3]
	resourceID := parts[5]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), resourceID)...)
}
