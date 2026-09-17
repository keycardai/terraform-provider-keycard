package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource                     = &UserDataSource{}
	_ datasource.DataSourceWithConfigValidators = &UserDataSource{}
)

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// UserDataSource defines the data source implementation.
type UserDataSource struct {
	client *client.ClientWithResponses
}

type UserModel struct {
	ID         types.String `tfsdk:"id"`
	ZoneID     types.String `tfsdk:"zone_id"`
	Identifier types.String `tfsdk:"identifier"`
	Email      types.String `tfsdk:"email"`
	Subject    types.String `tfsdk:"subject"`
	Issuer     types.String `tfsdk:"issuer"`
	Status     types.String `tfsdk:"status"`
	External   types.Bool   `tfsdk:"external"`
	ProviderID types.String `tfsdk:"provider_id"`
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resolves a Keycard user from a federated or SCIM identity so it can be referenced in role assignments and group memberships.\n\n" +
			"Exactly one lookup must be set: `id`, `identifier`, `email` with `issuer`, or `subject` with `issuer`. " +
			"The lookup must match exactly one user; zero or multiple matches are an error.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the user. Lookup key; conflicts with every other lookup attribute.",
				Optional:            true,
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "The zone this user belongs to.",
				Required:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Zone-scoped user identifier. Lookup key; conflicts with every other lookup attribute.",
				Optional:            true,
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address of the user. Lookup key; requires `issuer`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("issuer")),
				},
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "Subject identifier from the identity provider. Lookup key; requires `issuer`. `null` until a SCIM-created user first logs in.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("issuer")),
				},
			},
			"issuer": schema.StringAttribute{
				MarkdownDescription: "Issuer of the identity provider. Scopes an `email` or `subject` lookup. `null` until a SCIM-created user first logs in.",
				Optional:            true,
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the user: `active` or `disabled`.",
				Computed:            true,
			},
			"external": schema.BoolAttribute{
				MarkdownDescription: "Whether the user is synced from an external directory over SCIM.",
				Computed:            true,
			},
			"provider_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity provider the user authenticated through. `null` when the provider has been deleted.",
				Computed:            true,
			},
		},
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("identifier"),
			path.MatchRoot("email"),
			path.MatchRoot("subject"),
		),
		datasourcevalidator.Conflicting(
			path.MatchRoot("issuer"),
			path.MatchRoot("id"),
		),
		datasourcevalidator.Conflicting(
			path.MatchRoot("issuer"),
			path.MatchRoot("identifier"),
		),
	}
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var user *client.User
	if !data.ID.IsNull() {
		user = d.getUser(ctx, data, resp)
	} else {
		user = d.findUser(ctx, data, resp)
	}
	if user == nil {
		return
	}

	updateUserModelFromAPIResponse(user, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *UserDataSource) getUser(ctx context.Context, data UserModel, resp *datasource.ReadResponse) *client.User {
	getResp, err := d.client.GetUserWithResponse(ctx, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user, got error: %s", err))
		return nil
	}

	if getResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(
			"User Not Found",
			fmt.Sprintf("User with ID %s not found in zone %s", data.ID.ValueString(), data.ZoneID.ValueString()),
		)
		return nil
	}

	if getResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read user, got status %d: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return nil
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Unable to read user, no response body")
		return nil
	}

	return getResp.JSON200
}

// findUser lists users with exact filters from the configured lookup set and
// requires exactly one match.
func (d *UserDataSource) findUser(ctx context.Context, data UserModel, resp *datasource.ReadResponse) *client.User {
	// Fetch two rows so ambiguity is visible.
	limit := 2
	params := &client.ListUsersParams{Limit: &limit}
	var lookup []string

	switch {
	case !data.Identifier.IsNull():
		params.FilterIdentifier = &[]string{data.Identifier.ValueString()}
		lookup = append(lookup, fmt.Sprintf("identifier '%s'", data.Identifier.ValueString()))
	case !data.Email.IsNull():
		params.FilterEmail = &[]string{data.Email.ValueString()}
		lookup = append(lookup, fmt.Sprintf("email '%s'", data.Email.ValueString()))
	case !data.Subject.IsNull():
		params.FilterSubject = &[]string{data.Subject.ValueString()}
		lookup = append(lookup, fmt.Sprintf("subject '%s'", data.Subject.ValueString()))
	}
	if !data.Issuer.IsNull() {
		params.FilterIssuer = &[]string{data.Issuer.ValueString()}
		lookup = append(lookup, fmt.Sprintf("issuer '%s'", data.Issuer.ValueString()))
	}
	lookupDesc := strings.Join(lookup, " and ")

	listResp, err := d.client.ListUsersWithResponse(ctx, data.ZoneID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list users: %s", err))
		return nil
	}

	if listResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to list users, got status %d: %s", listResp.StatusCode(), string(listResp.Body)),
		)
		return nil
	}

	if listResp.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", "Received empty response from API")
		return nil
	}

	items := listResp.JSON200.Items
	if len(items) == 0 {
		resp.Diagnostics.AddError(
			"User Not Found",
			fmt.Sprintf("No user found with %s in zone '%s'", lookupDesc, data.ZoneID.ValueString()),
		)
		return nil
	}

	if len(items) > 1 {
		ids := make([]string, len(items))
		for i, u := range items {
			ids[i] = u.Id
		}
		resp.Diagnostics.AddError(
			"Multiple Users Found",
			fmt.Sprintf("Expected exactly 1 user with %s in zone '%s', but found matches: %s. Look the user up by `id` instead.",
				lookupDesc, data.ZoneID.ValueString(), strings.Join(ids, ", ")),
		)
		return nil
	}

	return &items[0]
}

func updateUserModelFromAPIResponse(user *client.User, data *UserModel) {
	data.ID = types.StringValue(user.Id)
	data.ZoneID = types.StringValue(user.ZoneId)
	data.Identifier = types.StringValue(user.Identifier)
	data.Email = types.StringValue(string(user.Email))
	data.Subject = types.StringPointerValue(user.Subject)
	data.Issuer = types.StringPointerValue(user.Issuer)
	data.Status = types.StringValue(string(user.Status))
	data.External = types.BoolValue(user.External)
	data.ProviderID = types.StringPointerValue(user.ProviderId)
}
