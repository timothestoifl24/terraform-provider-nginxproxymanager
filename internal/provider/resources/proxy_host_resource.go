// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &proxyHostResource{}
	_ resource.ResourceWithConfigure      = &proxyHostResource{}
	_ resource.ResourceWithImportState    = &proxyHostResource{}
	_ resource.ResourceWithValidateConfig = &proxyHostResource{}
)

type proxyHostResource struct {
	client *sdk.Client
}

func NewProxyHostResource() resource.Resource {
	return &proxyHostResource{}
}

func (r *proxyHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_proxy_host"
}

func (r *proxyHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the proxy host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forward_host": schema.StringAttribute{
				Required:    true,
				Description: "Host to forward the request on to.",
			},
			"forward_port": schema.Int64Attribute{
				Required:    true,
				Description: "Port to forward to.",
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"forward_scheme": schema.StringAttribute{
				Required:    true,
				Description: "Scheme to use when forwarding, `http` or `https`.",
				Validators: []validator.String{
					stringvalidator.OneOf("http", "https"),
				},
			},
			"domain_names": schema.SetAttribute{
				Required:    true,
				Description: "Domain names to accept. NPM stores these sorted, so the order doesn't matter.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"access_list_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "ID of the access list to use.",
				Default:     int64default.StaticInt64(0),
			},
			"certificate_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "ID of the certificate to use.",
				Default:     int64default.StaticInt64(0),
			},
			"ssl_forced": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to force SSL.",
				Default:     booldefault.StaticBool(false),
			},
			"caching_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable caching,",
				Default:     booldefault.StaticBool(false),
			},
			"block_exploits": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to block exploits",
				Default:     booldefault.StaticBool(false),
			},
			"allow_websocket_upgrade": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to allow websockets.",
				Default:     booldefault.StaticBool(false),
			},
			"http2_support": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable HTTP2 support.",
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable the proxy host.",
				Default:     booldefault.StaticBool(true),
			},
			"hsts_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable HSTS.",
				Default:     booldefault.StaticBool(false),
			},
			"hsts_subdomains": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable HSTS for subdomains.",
				Default:     booldefault.StaticBool(false),
			},
			"trust_forwarded_proto": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Trust the X-Forwarded-Proto header from an upstream proxy when forcing SSL.",
				Default:     booldefault.StaticBool(false),
			},
			"advanced_config": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Advanced NGINX config.",
				Default:     stringdefault.StaticString(""),
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user which owns the proxy host.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"nginx_online": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether nginx accepted the config generated for this host on the last change.",
			},
			"nginx_err": schema.StringAttribute{
				Computed:    true,
				Description: "The error nginx reported for this host's config, if any.",
			},
		},
		Blocks: map[string]schema.Block{
			"location": schema.ListNestedBlock{
				Description: "Custom locations, each forwarding a path to a different host.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Required:    true,
							Description: "Path to match, e.g `/api`.",
						},
						"forward_host": schema.StringAttribute{
							Required:    true,
							Description: "Host to forward the request on to.",
						},
						"forward_port": schema.Int64Attribute{
							Required:    true,
							Description: "Port to forward to.",
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"forward_scheme": schema.StringAttribute{
							Required:    true,
							Description: "Scheme to use when forwarding, `http` or `https`.",
							Validators: []validator.String{
								stringvalidator.OneOf("http", "https"),
							},
						},
						"forward_path": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Path on the forward host to send the request to.",
							Default:     stringdefault.StaticString(""),
						},
						"advanced_config": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Advanced NGINX config for this location.",
							Default:     stringdefault.StaticString(""),
						},
						"access_list_id": schema.Int64Attribute{
							Optional:    true,
							Computed:    true,
							Description: "ID of an access list for this location. Requires NPM 2.16.0 or later.",
							Default:     int64default.StaticInt64(0),
						},
					},
				},
			},
		},
	}
}

func (r *proxyHostResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.ProxyHostModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	validateSSLSettings(data.CertificateID, data.SSLForced, data.HTTP2Support, data.HSTSEnabled, data.HSTSSubdomains, &resp.Diagnostics)
}

func (r *proxyHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.ProxyHostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.ProxyHosts.Create(plan.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating proxy host.",
			fmt.Sprintf("Unable to create proxy host, unexpected error: %s", err.Error()),
		)
		return
	}

	// From here on the host exists, so it's always saved to state, even if
	// the remaining steps fail. Terraform then marks it as tainted.
	var state models.ProxyHostModel
	state.FromAPI(created)
	defer func() { resp.Diagnostics.Append(resp.State.Set(ctx, &state)...) }()

	if !applyEnabled(r.client.ProxyHosts, created.ID, true, plan.Enabled, "proxy host", &resp.Diagnostics) {
		return
	}

	host, err := r.client.ProxyHosts.Get(created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading proxy host after creating it.", err.Error())
		return
	}
	state.FromAPI(host)
	warnIfNginxOffline(state.NginxOnline, state.NginxErr, "proxy host", &resp.Diagnostics)
}

func (r *proxyHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.ProxyHostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	host, err := r.client.ProxyHosts.Get(id)
	if err != nil {
		handleReadError(ctx, resp, "proxy host", state.ID, err)
		return
	}

	state.FromAPI(host)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *proxyHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.ProxyHostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(plan.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	updated, err := r.client.ProxyHosts.Update(id, plan.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating proxy host.",
			fmt.Sprintf("Unable to update proxy host %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	enabled := updated.Enabled != nil && *updated.Enabled
	if !applyEnabled(r.client.ProxyHosts, id, enabled, plan.Enabled, "proxy host", &resp.Diagnostics) {
		return
	}

	host, err := r.client.ProxyHosts.Get(id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading proxy host after updating it.", err.Error())
		return
	}

	var state models.ProxyHostModel
	state.FromAPI(host)
	warnIfNginxOffline(state.NginxOnline, state.NginxErr, "proxy host", &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *proxyHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.ProxyHostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	err := r.client.ProxyHosts.Delete(id)
	if err != nil && !sdk.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting proxy host",
			fmt.Sprintf("Unable to delete proxy host, unexpected error: %s", err.Error()),
		)
	}
}

func (r *proxyHostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *proxyHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importNumericID(ctx, req, resp)
}
