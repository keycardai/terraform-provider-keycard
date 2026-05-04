package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/client"
)

var _ datasource.DataSource = &SSOLoginURLDataSource{}

func NewSSOLoginURLDataSource() datasource.DataSource {
	return &SSOLoginURLDataSource{}
}

// SSOLoginURLDataSource computes the IdP-initiated login URL for a given
// issuer. It resolves the organization ID from the configured service
// account credentials at plan time.
type SSOLoginURLDataSource struct {
	client *client.ClientWithResponses
}

type SSOLoginURLDataSourceModel struct {
	Issuer        types.String `tfsdk:"issuer"`
	TargetLinkURI types.String `tfsdk:"target_link_uri"`
	URL           types.String `tfsdk:"url"`
}

func (d *SSOLoginURLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_login_url"
}

func (d *SSOLoginURLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Computes the IdP-initiated login URL for an SSO connection. " +
			"Use this to configure the `login_uri` on your identity provider's OAuth app " +
			"(e.g., Okta) to enable IdP-initiated login to Keycard.",

		Attributes: map[string]schema.Attribute{
			"issuer": schema.StringAttribute{
				MarkdownDescription: "The OIDC issuer URL of the identity provider (e.g., `https://your-org.okta.com`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"target_link_uri": schema.StringAttribute{
				MarkdownDescription: "The URI the user is redirected to after login. Defaults to the Keycard Console.",
				Optional:            true,
				Computed:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The computed IdP-initiated login URL. Set this as the `login_uri` on your identity provider's OAuth app.",
				Computed:            true,
			},
		},
	}
}

func (d *SSOLoginURLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SSOLoginURLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SSOLoginURLDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := d.client.Endpoint()

	idURL, err := identityURL(endpoint)
	if err != nil {
		resp.Diagnostics.AddError("Configuration Error", fmt.Sprintf("Unable to derive identity URL from endpoint: %s", err))
		return
	}

	orgID, err := GetOrganizationID(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get organization ID: %s", err))
		return
	}

	targetLinkURI := data.TargetLinkURI.ValueString()
	if data.TargetLinkURI.IsNull() || data.TargetLinkURI.IsUnknown() {
		defaultURI, err := consoleURL(endpoint)
		if err != nil {
			resp.Diagnostics.AddError("Configuration Error", fmt.Sprintf("Unable to derive console URL from endpoint: %s", err))
			return
		}
		targetLinkURI = defaultURI
	}
	data.TargetLinkURI = types.StringValue(targetLinkURI)

	params := url.Values{}
	params.Set("iss", data.Issuer.ValueString())
	params.Set("target_link_uri", targetLinkURI)
	params.Set("tenant", orgID)

	data.URL = types.StringValue(idURL + "/openid/connect/login?" + params.Encode())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
