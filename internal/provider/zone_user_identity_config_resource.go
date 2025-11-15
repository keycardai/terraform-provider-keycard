package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
	"github.com/oapi-codegen/nullable"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ZoneUserIdentityConfigResource{}
	_ resource.ResourceWithImportState = &ZoneUserIdentityConfigResource{}
)

func NewZoneUserIdentityConfigResource() resource.Resource {
	return &ZoneUserIdentityConfigResource{}
}

// ZoneUserIdentityConfigResource defines the resource implementation.
type ZoneUserIdentityConfigResource struct {
	client *client.KeycardClient
}

// ZoneUserIdentityConfigResourceModel describes the resource data model.
type ZoneUserIdentityConfigResourceModel struct {
	ZoneID     types.String `tfsdk:"zone_id"`
	ProviderID types.String `tfsdk:"provider_id"`
}

func (r *ZoneUserIdentityConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_user_identity_config"
}

func (r *ZoneUserIdentityConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Configures the user identity provider for a Keycard zone. This resource manages which provider is used for user authentication in the zone.\n\n" +
			"~> **Note** Only one `keycard_zone_user_identity_config` resource should exist per zone. " +
			"Declaring multiple `keycard_zone_user_identity_config` resources for the same zone will cause a perpetual difference in configuration.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the zone to configure. Changing this will replace the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the provider to use for user authentication in this zone.",
				Required:            true,
			},
		},
	}
}

func (r *ZoneUserIdentityConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZoneUserIdentityConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneUserIdentityConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the update request to set the user identity provider
	updateReq := client.ZoneUpdate{
		UserIdentityProviderId: StringValueNullable(data.ProviderID),
	}

	// Update the zone to set the user identity provider
	updateResp, err := r.client.UpdateZoneWithResponse(ctx, data.ZoneID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to configure zone user identity provider, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to configure zone user identity provider, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to configure zone user identity provider, no response body")
		return
	}

	// Verify the configuration was applied
	zone := updateResp.JSON200
	if zone.UserIdentityProviderId == nil || *zone.UserIdentityProviderId != data.ProviderID.ValueString() {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Zone user identity provider was not set correctly. Expected %s, got %v", data.ProviderID.ValueString(), zone.UserIdentityProviderId),
		)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneUserIdentityConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneUserIdentityConfigResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the zone
	getResp, err := r.client.GetZoneWithResponse(ctx, data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read zone, got error: %s", err))
		return
	}

	if getResp.StatusCode() == 404 {
		// Zone was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read zone, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read zone, no response body")
		return
	}

	zone := getResp.JSON200

	// Check if the zone has a user identity provider configured
	if zone.UserIdentityProviderId == nil || *zone.UserIdentityProviderId == "" {
		// No provider configured on the zone - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Update provider_id from the zone's current configuration
	// This allows Terraform to detect drift if the provider was changed externally
	data.ProviderID = types.StringPointerValue(zone.UserIdentityProviderId)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneUserIdentityConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ZoneUserIdentityConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the update request to change the user identity provider
	updateReq := client.ZoneUpdate{
		UserIdentityProviderId: StringValueNullable(data.ProviderID),
	}

	// Update the zone
	updateResp, err := r.client.UpdateZoneWithResponse(ctx, data.ZoneID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update zone user identity provider, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update zone user identity provider, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to update zone user identity provider, no response body")
		return
	}

	// Verify the configuration was applied
	zone := updateResp.JSON200
	if zone.UserIdentityProviderId == nil || *zone.UserIdentityProviderId != data.ProviderID.ValueString() {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Zone user identity provider was not updated correctly. Expected %s, got %v", data.ProviderID.ValueString(), zone.UserIdentityProviderId),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneUserIdentityConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneUserIdentityConfigResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the update request to remove the user identity provider
	updateReq := client.ZoneUpdate{
		UserIdentityProviderId: nullable.NewNullNullable[string](),
	}

	// Update the zone to unset the user identity provider
	updateResp, err := r.client.UpdateZoneWithResponse(ctx, data.ZoneID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove zone user identity provider, got error: %s", err))
		return
	}

	if updateResp.StatusCode() != 200 && updateResp.StatusCode() != 404 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to remove zone user identity provider, got status %d: %s", updateResp.StatusCode(), string(updateResp.Body)),
		)
		return
	}
}

func (r *ZoneUserIdentityConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by zone_id - the import ID is the zone_id
	resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}
