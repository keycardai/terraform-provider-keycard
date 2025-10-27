package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Default endpoint to production API.
const defaultEndpoint = "https://api.keycard.ai"

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &ScaffoldingProvider{}
var _ provider.ProviderWithFunctions = &ScaffoldingProvider{}
var _ provider.ProviderWithEphemeralResources = &ScaffoldingProvider{}

// ScaffoldingProvider defines the provider implementation.
type ScaffoldingProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// KeycardProviderModel describes the provider data model.
type KeycardProviderModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	Endpoint       types.String `tfsdk:"endpoint"`
}

func (p *ScaffoldingProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "keycard"
	resp.Version = p.version
}

func (p *ScaffoldingProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Keycard provider is used to interact with Keycard resources. " +
			"The provider requires OAuth2 client credentials authentication to be configured.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The Keycard organization ID. Can also be set via the `KEYCARD_ORGANIZATION_ID` environment variable.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The OAuth2 client ID for authentication. Can also be set via the `KEYCARD_CLIENT_ID` environment variable.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The OAuth2 client secret for authentication. Can also be set via the `KEYCARD_CLIENT_SECRET` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Keycard API endpoint. Can also be set via the `KEYCARD_ENDPOINT` environment variable. Defaults to production API.",
				Optional:            true,
			},
		},
	}
}

func (p *ScaffoldingProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data KeycardProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read configuration values with environment variable fallback
	organizationID := data.OrganizationID.ValueString()
	if organizationID == "" {
		organizationID = os.Getenv("KEYCARD_ORGANIZATION_ID")
	}

	clientID := data.ClientID.ValueString()
	if clientID == "" {
		clientID = os.Getenv("KEYCARD_CLIENT_ID")
	}

	clientSecret := data.ClientSecret.ValueString()
	if clientSecret == "" {
		clientSecret = os.Getenv("KEYCARD_CLIENT_SECRET")
	}

	endpoint := data.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("KEYCARD_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	// Validate required parameters
	if organizationID == "" {
		resp.Diagnostics.AddError(
			"Missing Organization ID",
			"The provider cannot create the Keycard API client as there is a missing or empty value for the organization ID. "+
				"Set the organization_id value in the configuration or use the KEYCARD_ORGANIZATION_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if clientID == "" {
		resp.Diagnostics.AddError(
			"Missing Client ID",
			"The provider cannot create the Keycard API client as there is a missing or empty value for the client ID. "+
				"Set the client_id value in the configuration or use the KEYCARD_CLIENT_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if clientSecret == "" {
		resp.Diagnostics.AddError(
			"Missing Client Secret",
			"The provider cannot create the Keycard API client as there is a missing or empty value for the client secret. "+
				"Set the client_secret value in the configuration or use the KEYCARD_CLIENT_SECRET environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	// Create fully configured API client with OAuth2, retries, and logging
	apiClient, err := client.NewAPIClient(ctx, client.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		OrganizationID: organizationID,
		Endpoint:       endpoint,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Keycard API Client",
			"An unexpected error occurred when creating the Keycard API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Keycard Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

func (p *ScaffoldingProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewZoneResource,
		NewProviderResource,
		NewZoneUserIdentityConfigResource,
		NewApplicationResource,
		NewResourceResource,
		NewApplicationDependencyResource,
	}
}

func (p *ScaffoldingProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *ScaffoldingProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewZoneDataSource,
		NewProviderDataSource,
		NewZoneUserIdentityConfigDataSource,
		NewZoneDirectoryDataSource,
		NewApplicationDataSource,
		NewResourceDataSource,
	}
}

func (p *ScaffoldingProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ScaffoldingProvider{
			version: version,
		}
	}
}
