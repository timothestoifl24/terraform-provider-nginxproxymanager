// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dnsProvidersDataSource{}
	_ datasource.DataSourceWithConfigure = &dnsProvidersDataSource{}
)

type dnsProviderModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Credentials types.String `tfsdk:"credentials"`
}

type dnsProvidersModel struct {
	Providers []dnsProviderModel `tfsdk:"providers"`
}

type dnsProvidersDataSource struct {
	client *sdk.Client
}

func NewDNSProvidersDataSource() datasource.DataSource {
	return &dnsProvidersDataSource{}
}

func (d *dnsProvidersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_providers"
}

func (d *dnsProvidersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The DNS providers NPM supports for Letsencrypt DNS challenges.",
		Attributes: map[string]schema.Attribute{
			"providers": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The supported DNS providers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID to use as `dns_provider` on a Letsencrypt certificate.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the DNS provider.",
						},
						"credentials": schema.StringAttribute{
							Computed:    true,
							Description: "Example of the credentials the DNS provider expects in `dns_provider_credentials`.",
						},
					},
				},
			},
		},
	}
}

func (d *dnsProvidersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	providers, err := d.client.Certificates.DNSProviders()
	if err != nil {
		resp.Diagnostics.AddError("Unable to list DNS providers.", err.Error())
		return
	}

	var state dnsProvidersModel
	state.Providers = []dnsProviderModel{}
	for _, p := range providers {
		state.Providers = append(state.Providers, dnsProviderModel{
			ID:          types.StringValue(p.ID),
			Name:        types.StringValue(p.Name),
			Credentials: types.StringValue(p.Credentials),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *dnsProvidersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureProvider(req, resp)
}
