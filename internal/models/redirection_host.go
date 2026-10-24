// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strconv"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RedirectionHostModel struct {
	ID                types.String   `tfsdk:"id"`
	DomainNames       []types.String `tfsdk:"domain_names"`
	ForwardDomainName types.String   `tfsdk:"forward_domain_name"`
	ForwardHTTPCode   types.Int64    `tfsdk:"forward_http_code"`
	ForwardScheme     types.String   `tfsdk:"forward_scheme"`
	CertificateID     types.Int64    `tfsdk:"certificate_id"`
	SSLForced         types.Bool     `tfsdk:"ssl_forced"`
	BlockExploits     types.Bool     `tfsdk:"block_exploits"`
	AdvancedConfig    types.String   `tfsdk:"advanced_config"`
	HTTP2Support      types.Bool     `tfsdk:"http2_support"`
	Enabled           types.Bool     `tfsdk:"enabled"`
	HSTSEnabled       types.Bool     `tfsdk:"hsts_enabled"`
	HSTSSubdomains    types.Bool     `tfsdk:"hsts_subdomains"`
	PreservePath      types.Bool     `tfsdk:"preserve_path"`
	OwnerUserID       types.Int64    `tfsdk:"owner_user_id"`
}

// ToAPI returns the fields to send to NPM when creating or updating the
// redirection host. Enabled is set through the enable/disable endpoints.
func (m RedirectionHostModel) ToAPI() sdk.RedirectionHostBase {
	return sdk.RedirectionHostBase{
		ForwardScheme:     m.ForwardScheme.ValueStringPointer(),
		DomainNames:       returnPointer(StringValues(m.DomainNames)),
		ForwardDomainName: m.ForwardDomainName.ValueStringPointer(),
		ForwardHTTPCode:   returnPointer(int(m.ForwardHTTPCode.ValueInt64())),
		CertificateID:     returnPointer(int(m.CertificateID.ValueInt64())),
		SSLForced:         m.SSLForced.ValueBoolPointer(),
		BlockExploits:     m.BlockExploits.ValueBoolPointer(),
		AdvancedConfig:    m.AdvancedConfig.ValueStringPointer(),
		HTTP2Support:      m.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:       m.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains:    m.HSTSSubdomains.ValueBoolPointer(),
		PreservePath:      m.PreservePath.ValueBoolPointer(),
	}
}

// FromAPI sets the model from a redirection host returned by NPM.
func (m *RedirectionHostModel) FromAPI(host sdk.RedirectionHost) {
	m.ID = types.StringValue(strconv.Itoa(host.ID))
	m.OwnerUserID = types.Int64Value(int64(host.OwnerUserID))
	m.DomainNames = StringsValue(derefSlice(host.DomainNames))
	m.ForwardScheme = types.StringPointerValue(host.ForwardScheme)
	m.ForwardDomainName = types.StringPointerValue(host.ForwardDomainName)
	m.ForwardHTTPCode = intPointerValue(host.ForwardHTTPCode)
	m.CertificateID = intValueOrZero(host.CertificateID)
	m.SSLForced = boolValueOrFalse(host.SSLForced)
	m.BlockExploits = boolValueOrFalse(host.BlockExploits)
	m.AdvancedConfig = stringValueOrEmpty(host.AdvancedConfig)
	m.HTTP2Support = boolValueOrFalse(host.HTTP2Support)
	m.Enabled = boolValueOrFalse(host.Enabled)
	m.HSTSEnabled = boolValueOrFalse(host.HSTSEnabled)
	m.HSTSSubdomains = boolValueOrFalse(host.HSTSSubdomains)
	m.PreservePath = boolValueOrFalse(host.PreservePath)
}
