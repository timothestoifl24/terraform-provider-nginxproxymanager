// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	datasources "github.com/edklesel/terraform-provider-nginxproxymanager/internal/provider/datasources"
	resources "github.com/edklesel/terraform-provider-nginxproxymanager/internal/provider/resources"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure npmProvider satisfies various provider interfaces.
var (
	_ provider.Provider                       = &npmProvider{}
	_ provider.ProviderWithFunctions          = &npmProvider{}
	_ provider.ProviderWithEphemeralResources = &npmProvider{}

	minVersion = "2.15.0"
)

// npmProvider defines the provider implementation.
type npmProvider struct {
	version string
}

// npmProviderModel describes the provider data model.
type npmProviderModel struct {
	Host           types.String `tfsdk:"host"`
	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	Token          types.String `tfsdk:"token"`
	Secure         types.Bool   `tfsdk:"secure"`
	RequestTimeout types.Int64  `tfsdk:"request_timeout"`
}

func (p *npmProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nginxproxymanager"
	resp.Version = p.version
}

func (p *npmProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "Hostname and port of Nginx Proxy Manager, e.g `npm.example.com:81`. Can also be set with `NGINXPROXYMANAGER_HOST`.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Email address of the user to authenticate as. Can also be set with `NGINXPROXYMANAGER_USERNAME`.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password of the user to authenticate as. Can also be set with `NGINXPROXYMANAGER_PASSWORD`.",
				Optional:            true,
				Sensitive:           true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "An API token to use instead of logging in, e.g for a user with two-factor authentication. " +
					"A long-lived one can be requested with `GET /api/tokens?expiry=1y` while logged in. " +
					"The token is refreshed while it's valid, and `username`/`password` are used to log in again if it expires. " +
					"Can also be set with `NGINXPROXYMANAGER_TOKEN`.",
				Optional:  true,
				Sensitive: true,
			},
			"secure": schema.BoolAttribute{
				MarkdownDescription: "Whether to use a secure (https) connection for API requests. Can also be set with `NGINXPROXYMANAGER_SECURE`.",
				Optional:            true,
			},
			"request_timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for a single API request. Defaults to 1200, as NPM answers certificate requests only once Letsencrypt has issued the certificate.",
				Optional:            true,
			},
		},
	}
}

// stringFromConfigOrEnv returns the configured value, or the environment
// variable when it isn't configured.
func stringFromConfigOrEnv(value types.String, env string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}
	return os.Getenv(env)
}

func (p *npmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

	tflog.Info(ctx, "Configuring Nginx Proxy Manager client.")

	var data npmProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	for attr, value := range map[string]attr.Value{
		"host":     data.Host,
		"username": data.Username,
		"password": data.Password,
		"token":    data.Token,
	} {
		if value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root(attr),
				fmt.Sprintf("Unknown Nginx Proxy Manager %s", attr),
				fmt.Sprintf("The '%s' parameter must be known when the provider is configured.", attr),
			)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	host := stringFromConfigOrEnv(data.Host, "NGINXPROXYMANAGER_HOST")
	username := stringFromConfigOrEnv(data.Username, "NGINXPROXYMANAGER_USERNAME")
	password := stringFromConfigOrEnv(data.Password, "NGINXPROXYMANAGER_PASSWORD")
	token := stringFromConfigOrEnv(data.Token, "NGINXPROXYMANAGER_TOKEN")

	secure := data.Secure.ValueBool()
	if data.Secure.IsNull() {
		secure, _ = strconv.ParseBool(os.Getenv("NGINXPROXYMANAGER_SECURE"))
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Nginx Proxy Manager host",
			"Set 'host' in the provider configuration or the NGINXPROXYMANAGER_HOST environment variable.",
		)
	}
	if token == "" && (username == "" || password == "") {
		resp.Diagnostics.AddError(
			"Missing Nginx Proxy Manager credentials",
			"Set 'username' and 'password', or 'token', in the provider configuration or with the "+
				"NGINXPROXYMANAGER_USERNAME, NGINXPROXYMANAGER_PASSWORD and NGINXPROXYMANAGER_TOKEN environment variables.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var protocol = "http"
	if secure {
		protocol = "https"
	}

	timeout := 20 * time.Minute
	if !data.RequestTimeout.IsNull() {
		timeout = time.Duration(data.RequestTimeout.ValueInt64()) * time.Second
	}

	ctx = tflog.SetField(ctx, "nginx_proxy_manager_host", host)
	ctx = tflog.SetField(ctx, "nginx_proxy_manager_username", username)
	ctx = tflog.SetField(ctx, "nginx_proxy_manager_secure", secure)

	tflog.Debug(ctx, "Creating Nginx Proxy Manager client.")

	client, err := sdk.NewClientWithOptions(
		fmt.Sprintf("%s://%s", protocol, host),
		sdk.AuthStruct{
			Identity: username,
			Secret:   password,
		},
		sdk.ClientOptions{
			Token:   token,
			Timeout: timeout,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to connect to Nginx Proxy Manager",
			err.Error(),
		)
		return
	}

	// Check if NPM is the minimum version
	versionConstraint, err := version.NewConstraint(fmt.Sprintf(">= %s", minVersion))
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed checking the minimum version",
			fmt.Sprintf(
				"The following error occurred while checking the minimum version: %s",
				err.Error(),
			),
		)
		return
	}
	if !versionConstraint.Check(client.Version) {
		resp.Diagnostics.AddError(
			"Minimum version requirements not met",
			fmt.Sprintf(
				"The NGINX Proxy Manager Terraform Provider requires version %s or above, found %s.",
				minVersion, client.Version,
			),
		)
		return
	}

	// Log in now rather than on the first request, so wrong credentials are
	// reported as a provider error. A configured token is checked on first use.
	if token == "" {
		tflog.Info(ctx, "Client created, fetching token.")

		err = client.GetToken()
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed creating Nginx Proxy Manager client",
				fmt.Sprintf(
					"The following error occurred while creating the Nginx Proxy Manager client: %s",
					err.Error(),
				),
			)
			return
		}
	}

	tflog.Info(ctx, "Configured Nginx Proxy Manager client.", map[string]any{"success": true})

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *npmProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewUserResource,
		resources.NewUserPasswordResource,
		resources.NewUserPermissionsResource,
		resources.NewAccessListResource,
		resources.NewProxyHostResource,
		resources.NewDeadHostResource,
		resources.NewRedirectionHostResource,
		resources.NewStreamResource,
		resources.NewCustomCertificateResource,
		resources.NewLetsencryptCertificateResource,
		resources.NewDefaultSiteResource,
	}
}

func (p *npmProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *npmProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewUserDataSource,
		datasources.NewUsersDataSource,
		datasources.NewProxyHostDataSource,
		datasources.NewAccessListDataSource,
		datasources.NewDeadHostDataSource,
		datasources.NewRedirectionHostDataSource,
		datasources.NewStreamDataSource,
		datasources.NewCustomCertificateDataSource,
		datasources.NewletsencryptCertificateDataSource,
		datasources.NewDNSProvidersDataSource,
		datasources.NewVersionDataSource,
	}
}

func (p *npmProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &npmProvider{
			version: version,
		}
	}
}
