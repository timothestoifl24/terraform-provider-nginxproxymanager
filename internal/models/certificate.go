// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strconv"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type LetsencryptCertificateModel struct {
	ID                     types.String   `tfsdk:"id"`
	DomainNames            []types.String `tfsdk:"domain_names"`
	DNSChallenge           types.Bool     `tfsdk:"dns_challenge"`
	DNSProvider            types.String   `tfsdk:"dns_provider"`
	DNSProviderCredentials types.String   `tfsdk:"dns_provider_credentials"`
	LetsencryptEmail       types.String   `tfsdk:"letsencrypt_email"`
	PropagationSeconds     types.Int64    `tfsdk:"propagation_seconds"`
	NiceName               types.String   `tfsdk:"nice_name"`
	// From                   types.Int64    `tfsdk:"from"`
	// To                     types.Int64    `tfsdk:"to"`
	// CN                     types.String   `tfsdk:"cn"`
	// Issuer                 types.String   `tfsdk:"issuer"`
}

// FromAPI sets the model from a certificate returned by NPM. The DNS provider
// credentials and the Letsencrypt email aren't returned by NPM, so they're
// left as they are.
func (m *LetsencryptCertificateModel) FromAPI(cert sdk.Certificate) {
	m.ID = types.StringValue(strconv.Itoa(cert.ID))
	m.NiceName = stringValueOrEmpty(cert.NiceName)
	m.DomainNames = StringsValue(derefSlice(cert.DomainNames))

	meta := cert.Meta
	if meta == nil {
		meta = &sdk.CertificateMeta{}
	}
	m.DNSChallenge = boolValueOrFalse(meta.DNSChallenge)
	m.DNSProvider = types.StringNull()
	if meta.DNSProvider != nil && *meta.DNSProvider != "" {
		m.DNSProvider = types.StringValue(*meta.DNSProvider)
	}
	m.PropagationSeconds = intPointerValue(meta.PropagationSeconds)
}

type CustomCertificateModel struct {
	ID                      types.String `tfsdk:"id"`
	NiceName                types.String `tfsdk:"nice_name"`
	Certificate             types.String `tfsdk:"certificate"`
	CertificateKey          types.String `tfsdk:"certificate_key"`
	IntermediateCertificate types.String `tfsdk:"intermediate_certificate"`
}

// FromAPI sets the model from a certificate returned by NPM. Values NPM
// doesn't return are left as they are.
func (m *CustomCertificateModel) FromAPI(cert sdk.Certificate) {
	m.ID = types.StringValue(strconv.Itoa(cert.ID))
	m.NiceName = stringValueOrEmpty(cert.NiceName)
	if cert.Meta == nil {
		return
	}
	if cert.Meta.Certificate != nil {
		m.Certificate = types.StringValue(*cert.Meta.Certificate)
	}
	if cert.Meta.CertificateKey != nil {
		m.CertificateKey = types.StringValue(*cert.Meta.CertificateKey)
	}
}
