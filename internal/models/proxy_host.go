// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strconv"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProxyHostLocationModel struct {
	ForwardScheme  types.String `tfsdk:"forward_scheme"`
	ForwardHost    types.String `tfsdk:"forward_host"`
	ForwardPort    types.Int64  `tfsdk:"forward_port"`
	ForwardPath    types.String `tfsdk:"forward_path"`
	Path           types.String `tfsdk:"path"`
	AdvancedConfig types.String `tfsdk:"advanced_config"`
	AccessListID   types.Int64  `tfsdk:"access_list_id"`
}

type ProxyHostModel struct {
	ID                    types.String             `tfsdk:"id"`
	ForwardScheme         types.String             `tfsdk:"forward_scheme"`
	DomainNames           []types.String           `tfsdk:"domain_names"`
	ForwardHost           types.String             `tfsdk:"forward_host"`
	ForwardPort           types.Int64              `tfsdk:"forward_port"`
	AccessListID          types.Int64              `tfsdk:"access_list_id"`
	CertificateID         types.Int64              `tfsdk:"certificate_id"`
	SSLForced             types.Bool               `tfsdk:"ssl_forced"`
	CachingEnabled        types.Bool               `tfsdk:"caching_enabled"`
	BlockExploits         types.Bool               `tfsdk:"block_exploits"`
	AdvancedConfig        types.String             `tfsdk:"advanced_config"`
	AllowWebsocketUpgrade types.Bool               `tfsdk:"allow_websocket_upgrade"`
	HTTP2Support          types.Bool               `tfsdk:"http2_support"`
	Enabled               types.Bool               `tfsdk:"enabled"`
	Locations             []ProxyHostLocationModel `tfsdk:"location"`
	HSTSEnabled           types.Bool               `tfsdk:"hsts_enabled"`
	HSTSSubdomains        types.Bool               `tfsdk:"hsts_subdomains"`
	TrustForwardedProto   types.Bool               `tfsdk:"trust_forwarded_proto"`
	OwnerUserID           types.Int64              `tfsdk:"owner_user_id"`
	NginxOnline           types.Bool               `tfsdk:"nginx_online"`
	NginxErr              types.String             `tfsdk:"nginx_err"`
}

// ToAPI returns the fields to send to NPM when creating or updating the
// proxy host. Enabled is left out because NPM only applies it through the
// enable/disable endpoints.
func (m ProxyHostModel) ToAPI() sdk.ProxyHostBase {
	host := sdk.ProxyHostBase{
		ForwardScheme:         m.ForwardScheme.ValueStringPointer(),
		ForwardHost:           m.ForwardHost.ValueStringPointer(),
		ForwardPort:           returnPointer(int(m.ForwardPort.ValueInt64())),
		DomainNames:           returnPointer(StringValues(m.DomainNames)),
		AccessListID:          returnPointer(int(m.AccessListID.ValueInt64())),
		CertificateID:         returnPointer(int(m.CertificateID.ValueInt64())),
		SSLForced:             m.SSLForced.ValueBoolPointer(),
		CachingEnabled:        m.CachingEnabled.ValueBoolPointer(),
		BlockExploits:         m.BlockExploits.ValueBoolPointer(),
		AdvancedConfig:        m.AdvancedConfig.ValueStringPointer(),
		AllowWebsocketUpgrade: m.AllowWebsocketUpgrade.ValueBoolPointer(),
		HTTP2Support:          m.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:           m.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains:        m.HSTSSubdomains.ValueBoolPointer(),
		TrustForwardedProto:   m.TrustForwardedProto.ValueBoolPointer(),
		Locations:             returnPointer([]sdk.Location{}),
	}

	for _, location := range m.Locations {
		_location := sdk.Location{
			ForwardScheme:  location.ForwardScheme.ValueString(),
			ForwardHost:    location.ForwardHost.ValueString(),
			ForwardPort:    int(location.ForwardPort.ValueInt64()),
			ForwardPath:    location.ForwardPath.ValueString(),
			Path:           location.Path.ValueString(),
			AdvancedConfig: location.AdvancedConfig.ValueString(),
		}
		if location.AccessListID.ValueInt64() != 0 {
			_location.AccessListID = returnPointer(int(location.AccessListID.ValueInt64()))
		}
		*host.Locations = append(*host.Locations, _location)
	}

	return host
}

// FromAPI sets the model from a proxy host returned by NPM.
func (m *ProxyHostModel) FromAPI(host sdk.ProxyHost) {
	m.ID = types.StringValue(strconv.Itoa(host.ID))
	m.OwnerUserID = types.Int64Value(int64(host.OwnerUserID))
	m.ForwardScheme = types.StringPointerValue(host.ForwardScheme)
	m.DomainNames = StringsValue(derefSlice(host.DomainNames))
	m.ForwardHost = types.StringPointerValue(host.ForwardHost)
	m.ForwardPort = intPointerValue(host.ForwardPort)
	m.AccessListID = intValueOrZero(host.AccessListID)
	m.CertificateID = intValueOrZero(host.CertificateID)
	m.SSLForced = boolValueOrFalse(host.SSLForced)
	m.AdvancedConfig = stringValueOrEmpty(host.AdvancedConfig)
	m.CachingEnabled = boolValueOrFalse(host.CachingEnabled)
	m.BlockExploits = boolValueOrFalse(host.BlockExploits)
	m.AllowWebsocketUpgrade = boolValueOrFalse(host.AllowWebsocketUpgrade)
	m.HTTP2Support = boolValueOrFalse(host.HTTP2Support)
	m.Enabled = boolValueOrFalse(host.Enabled)
	m.HSTSEnabled = boolValueOrFalse(host.HSTSEnabled)
	m.HSTSSubdomains = boolValueOrFalse(host.HSTSSubdomains)
	m.TrustForwardedProto = boolValueOrFalse(host.TrustForwardedProto)
	m.NginxOnline, m.NginxErr = hostMetaValues(host.Meta)

	m.Locations = []ProxyHostLocationModel{}
	for _, location := range derefSlice(host.Locations) {
		m.Locations = append(m.Locations, ProxyHostLocationModel{
			ForwardScheme:  types.StringValue(location.ForwardScheme),
			ForwardHost:    types.StringValue(location.ForwardHost),
			ForwardPort:    types.Int64Value(int64(location.ForwardPort)),
			ForwardPath:    types.StringValue(location.ForwardPath),
			Path:           types.StringValue(location.Path),
			AdvancedConfig: types.StringValue(location.AdvancedConfig),
			AccessListID:   intValueOrZero(location.AccessListID),
		})
	}
}
