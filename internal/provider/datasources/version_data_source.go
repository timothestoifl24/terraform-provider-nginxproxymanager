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
	_ datasource.DataSource              = &versionDataSource{}
	_ datasource.DataSourceWithConfigure = &versionDataSource{}
)

type versionModel struct {
	Version  types.String `tfsdk:"version"`
	Major    types.Int64  `tfsdk:"major"`
	Minor    types.Int64  `tfsdk:"minor"`
	Revision types.Int64  `tfsdk:"revision"`
}

type versionDataSource struct {
	client *sdk.Client
}

func NewVersionDataSource() datasource.DataSource {
	return &versionDataSource{}
}

func (d *versionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_version"
}

func (d *versionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The version of the running Nginx Proxy Manager.",
		Attributes: map[string]schema.Attribute{
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "Full version, e.g `2.16.0`.",
			},
			"major": schema.Int64Attribute{
				Computed:    true,
				Description: "Major version.",
			},
			"minor": schema.Int64Attribute{
				Computed:    true,
				Description: "Minor version.",
			},
			"revision": schema.Int64Attribute{
				Computed:    true,
				Description: "Revision.",
			},
		},
	}
}

func (d *versionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ver, err := d.client.GetVersion()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the NPM version.", err.Error())
		return
	}

	segments := ver.Segments64()
	state := versionModel{
		Version:  types.StringValue(ver.String()),
		Major:    types.Int64Value(segments[0]),
		Minor:    types.Int64Value(segments[1]),
		Revision: types.Int64Value(segments[2]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *versionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureProvider(req, resp)
}
