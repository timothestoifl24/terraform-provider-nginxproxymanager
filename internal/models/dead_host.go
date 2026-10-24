// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strconv"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DeadHostModel struct {
	ID             types.String   `tfsdk:"id"`
	DomainNames    []types.String `tfsdk:"domain_names"`
	CertificateID  types.Int64    `tfsdk:"certificate_id"`
	HTTP2Support   types.Bool     `tfsdk:"http2_support"`
	HSTSEnabled    types.Bool     `tfsdk:"hsts_enabled"`
	HSTSSubdomains types.Bool     `tfsdk:"hsts_subdomains"`
	SSLForced      types.Bool     `tfsdk:"ssl_forced"`
	OwnerUserID    types.Int64    `tfsdk:"owner_user_id"`
	AdvancedConfig types.String   `tfsdk:"advanced_config"`
	Enabled        types.Bool     `tfsdk:"enabled"`
}

// ToAPI returns the fields to send to NPM when creating or updating the
// dead host. Enabled is set through the enable/disable endpoints.
func (m DeadHostModel) ToAPI() sdk.DeadHostBase {
	return sdk.DeadHostBase{
		DomainNames:    returnPointer(StringValues(m.DomainNames)),
		AdvancedConfig: m.AdvancedConfig.ValueStringPointer(),
		HTTP2Support:   m.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:    m.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains: m.HSTSSubdomains.ValueBoolPointer(),
		SSLForced:      m.SSLForced.ValueBoolPointer(),
		CertificateID:  returnPointer(int(m.CertificateID.ValueInt64())),
	}
}

// FromAPI sets the model from a dead host returned by NPM.
func (m *DeadHostModel) FromAPI(host sdk.DeadHost) {
	m.ID = types.StringValue(strconv.Itoa(host.ID))
	m.OwnerUserID = types.Int64Value(int64(host.OwnerUserID))
	m.DomainNames = StringsValue(derefSlice(host.DomainNames))
	m.CertificateID = intValueOrZero(host.CertificateID)
	m.HTTP2Support = boolValueOrFalse(host.HTTP2Support)
	m.HSTSEnabled = boolValueOrFalse(host.HSTSEnabled)
	m.HSTSSubdomains = boolValueOrFalse(host.HSTSSubdomains)
	m.SSLForced = boolValueOrFalse(host.SSLForced)
	m.AdvancedConfig = stringValueOrEmpty(host.AdvancedConfig)
	m.Enabled = boolValueOrFalse(host.Enabled)
}
