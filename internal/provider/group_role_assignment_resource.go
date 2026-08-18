package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                     = &GroupRoleAssignmentResource{}
	_ resource.ResourceWithImportState      = &GroupRoleAssignmentResource{}
	_ resource.ResourceWithConfigValidators = &GroupRoleAssignmentResource{}
)

func NewGroupRoleAssignmentResource() resource.Resource {
	return &GroupRoleAssignmentResource{}
}

// GroupRoleAssignmentResource defines the resource implementation.
type GroupRoleAssignmentResource struct {
	client *client.ClientWithResponses
}

// GroupRoleAssignmentModel describes the resource data model.
type GroupRoleAssignmentModel struct {
	ID            types.String `tfsdk:"id"`
	ZoneID        types.String `tfsdk:"zone_id"`
	GroupID       types.String `tfsdk:"group_id"`
	RoleID        types.String `tfsdk:"role_id"`
	ScopeType     types.String `tfsdk:"scope_type"`
	ScopeID       types.String `tfsdk:"scope_id"`
	PrincipalType types.String `tfsdk:"principal_type"`
	PrincipalID   types.String `tfsdk:"principal_id"`
}

func (r *GroupRoleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_role_assignment"
}

func (r *GroupRoleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Assigns a role to a Keycard group. Every member of the group inherits the role. Use the `keycard_role` data source to look a role up by identifier and owner type.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the role assignment.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone the group belongs to. Changing this will replace the assignment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The group the role is assigned to. Changing this will replace the assignment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				MarkdownDescription: "ID of the role to assign. Changing this will replace the assignment.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_type": schema.StringAttribute{
				MarkdownDescription: "The kind of resource to scope the grant to, for example `zone`. Set together with `scope_id`, or omit both for an unscoped assignment. Changing this will replace the assignment.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the resource to scope the grant to. Set together with `scope_type`. Changing this will replace the assignment.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_type": schema.StringAttribute{
				MarkdownDescription: "The kind of principal the role is assigned to. Always `group` for this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"principal_id": schema.StringAttribute{
				MarkdownDescription: "ID of the principal the role is assigned to, matching `group_id`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *GroupRoleAssignmentResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.RequiredTogether(
			path.MatchRoot("scope_type"),
			path.MatchRoot("scope_id"),
		),
	}
}

func (r *GroupRoleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupRoleAssignmentModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.RoleAssignmentCreate{
		RoleId: data.RoleID.ValueStringPointer(),
	}

	if !data.ScopeType.IsNull() && !data.ScopeType.IsUnknown() {
		createReq.ScopeType = data.ScopeType.ValueStringPointer()
		createReq.ScopeId = data.ScopeID.ValueStringPointer()
	}

	createResp, err := r.client.AssignGroupRoleWithResponse(
		ctx,
		data.ZoneID.ValueString(),
		data.GroupID.ValueString(),
		createReq,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to assign group role, got error: %s", err))
		return
	}

	if createResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to assign group role, got status %d: %s", createResp.StatusCode(), string(createResp.Body)),
		)
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to assign group role, no response body")
		return
	}

	updateGroupRoleAssignmentModelFromAPIResponse(createResp.JSON201, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupRoleAssignmentModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// There is no endpoint for a single assignment, so filter the assignment
	// list down to the one we track.
	assignmentID := data.ID.ValueString()
	listResp, err := r.client.ListGroupRoleAssignmentsWithResponse(
		ctx,
		data.ZoneID.ValueString(),
		data.GroupID.ValueString(),
		&client.ListGroupRoleAssignmentsParams{FilterId: &[]string{assignmentID}},
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group role assignment, got error: %s", err))
		return
	}

	if listResp.StatusCode() == 404 {
		// The group itself is gone, so the assignment is too.
		resp.State.RemoveResource(ctx)
		return
	}

	if listResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read group role assignment, got status %d: %s", listResp.StatusCode(), string(listResp.Body)),
		)
		return
	}

	if listResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read group role assignment, no response body")
		return
	}

	for _, assignment := range listResp.JSON200.Items {
		if assignment.Id == assignmentID {
			updateGroupRoleAssignmentModelFromAPIResponse(&assignment, &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	// Assignment was revoked outside of Terraform
	resp.State.RemoveResource(ctx)
}

func (r *GroupRoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All configurable fields are immutable (RequiresReplace), so this should
	// never be called. If it is called, we just need to read the plan and set
	// it as the new state.
	var data GroupRoleAssignmentModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupRoleAssignmentModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Revoking a scoped grant requires the same scope pair it was created with.
	params := &client.RevokeGroupRoleParams{}
	if !data.ScopeType.IsNull() && !data.ScopeType.IsUnknown() {
		params.ScopeType = data.ScopeType.ValueStringPointer()
		params.ScopeId = data.ScopeID.ValueStringPointer()
	}

	deleteResp, err := r.client.RevokeGroupRoleWithResponse(
		ctx,
		data.ZoneID.ValueString(),
		data.GroupID.ValueString(),
		data.RoleID.ValueString(),
		params,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to revoke group role, got error: %s", err))
		return
	}

	// Accept both 204 (revoked) and 404 (already gone) as success
	if deleteResp.StatusCode() != 204 && deleteResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to revoke group role, got status %d: %s", deleteResp.StatusCode(), string(deleteResp.Body)),
		)
		return
	}
}

func (r *GroupRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID as zones/{zone-id}/groups/{group-id}/roles/{assignment-id}.
	// The assignment ID is used rather than the role ID because it is the only
	// value that uniquely identifies a scoped grant.
	parts := strings.Split(req.ID, "/")
	if len(parts) != 6 || parts[0] != "zones" || parts[2] != "groups" || parts[4] != "roles" ||
		parts[1] == "" || parts[3] == "" || parts[5] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'zones/{zone-id}/groups/{group-id}/roles/{assignment-id}', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[5])...)
}

// updateGroupRoleAssignmentModelFromAPIResponse maps a RoleAssignment API
// response onto the GroupRoleAssignmentModel.
func updateGroupRoleAssignmentModelFromAPIResponse(apiAssignment *client.RoleAssignment, data *GroupRoleAssignmentModel) {
	data.ID = types.StringValue(apiAssignment.Id)
	data.ZoneID = types.StringValue(apiAssignment.ZoneId)
	data.RoleID = types.StringValue(apiAssignment.RoleId)
	data.ScopeType = nullableStringValue(apiAssignment.ScopeType)
	data.ScopeID = nullableStringValue(apiAssignment.ScopeId)
	data.PrincipalType = types.StringValue(apiAssignment.PrincipalType)
	data.PrincipalID = types.StringValue(apiAssignment.PrincipalId)
}
