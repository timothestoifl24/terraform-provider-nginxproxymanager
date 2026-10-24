// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package go_npm_proxy_manager

type Location struct {
	ID             int    `json:"id"`
	ForwardScheme  string `json:"forward_scheme" proto:"http"`
	ForwardHost    string `json:"forward_host"`
	ForwardPort    int    `json:"forward_port" proto:"80"`
	ForwardPath    string `json:"forward_path"`
	Path           string `json:"path"`
	AdvancedConfig string `json:"advanced_config"`
	// Only supported from NPM 2.16.0, older versions reject it as an unknown field
	AccessListID *int `json:"access_list_id,omitempty"`
}

type ProxyHost struct {
	ProxyHostBase
	ID          int       `json:"id"`
	CreatedOn   string    `json:"created_on" deep:"-"`
	ModifiedOn  string    `json:"modified_on" deep:"-"`
	OwnerUserID int       `json:"owner_user_id"`
	Meta        *HostMeta `json:"meta,omitempty" deep:"-"`
	Enabled     *bool     `json:"enabled,omitempty"`
}

type ProxyHostBase struct {
	ForwardScheme         *string     `json:"forward_scheme,omitempty" client:"required" proto:"http"`
	DomainNames           *[]string   `json:"domain_names,omitempty" client:"required"`
	ForwardHost           *string     `json:"forward_host,omitempty" client:"required"`
	ForwardPort           *int        `json:"forward_port,omitempty" client:"required" proto:"80"`
	AccessListID          *int        `json:"access_list_id,omitempty"`
	CertificateID         *int        `json:"certificate_id,omitempty"`
	SSLForced             *bool       `json:"ssl_forced,omitempty"`
	CachingEnabled        *bool       `json:"caching_enabled,omitempty"`
	BlockExploits         *bool       `json:"block_exploits,omitempty"`
	AdvancedConfig        *string     `json:"advanced_config,omitempty"`
	AllowWebsocketUpgrade *bool       `json:"allow_websocket_upgrade,omitempty"`
	HTTP2Support          *bool       `json:"http2_support,omitempty"`
	Locations             *[]Location `json:"locations,omitempty" proto:"80"`
	HSTSEnabled           *bool       `json:"hsts_enabled,omitempty"`
	HSTSSubdomains        *bool       `json:"hsts_subdomains,omitempty"`
	TrustForwardedProto   *bool       `json:"trust_forwarded_proto,omitempty"`
}
