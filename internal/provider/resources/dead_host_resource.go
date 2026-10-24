// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	_ resource.Resource                = &deadHostResource{}
	_ resource.ResourceWithConfigure   = &deadHostResource{}
	_ resource.ResourceWithImportState = &deadHostResource{}
)

type deadHostResource struct {
	client *sdk.Client
}

func NewDeadHostResource() resource.Resource {
	return &deadHostResource{}
}

func (r *deadHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dead_host"
}

func (r *deadHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the dead host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"http2_support": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable HTTP2 support.",
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable the dead host.",
				Default:     booldefault.StaticBool(true),
			},
			"hsts_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable HSTS.",
				Validators:  []validator.Bool{},
				Default:     booldefault.StaticBool(false),
			},
			"hsts_subdomains": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable HSTS for subdomains.",
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
				Description: "ID of the user which owns the dead host.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *deadHostResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.DeadHostModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	validateSSLSettings(data.CertificateID, data.SSLForced, data.HTTP2Support, data.HSTSEnabled, data.HSTSSubdomains, &resp.Diagnostics)
}

func (r *deadHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DeadHostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.DeadHosts.Create(plan.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating dead host.",
			fmt.Sprintf("Unable to create dead host, unexpected error: %s", err.Error()),
		)
		return
	}

	// From here on the dead host exists, so it's always saved to state, even if
	// the remaining steps fail. Terraform then marks it as tainted.
	var state models.DeadHostModel
	state.FromAPI(created)
	defer func() { resp.Diagnostics.Append(resp.State.Set(ctx, &state)...) }()

	if !applyEnabled(r.client.DeadHosts, created.ID, true, plan.Enabled, "dead host", &resp.Diagnostics) {
		return
	}

	obj, err := r.client.DeadHosts.Get(created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dead host after creating it.", err.Error())
		return
	}
	state.FromAPI(obj)
	warnIfNginxOfflineMeta(obj.Meta, "dead host", &resp.Diagnostics)
}

func (r *deadHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DeadHostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	obj, err := r.client.DeadHosts.Get(id)
	if err != nil {
		handleReadError(ctx, resp, "dead host", state.ID, err)
		return
	}

	state.FromAPI(obj)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *deadHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DeadHostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(plan.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	updated, err := r.client.DeadHosts.Update(id, plan.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating dead host.",
			fmt.Sprintf("Unable to update dead host %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	enabled := updated.Enabled != nil && *updated.Enabled
	if !applyEnabled(r.client.DeadHosts, id, enabled, plan.Enabled, "dead host", &resp.Diagnostics) {
		return
	}

	obj, err := r.client.DeadHosts.Get(id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dead host after updating it.", err.Error())
		return
	}

	var state models.DeadHostModel
	state.FromAPI(obj)
	warnIfNginxOfflineMeta(obj.Meta, "dead host", &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *deadHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DeadHostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	err := r.client.DeadHosts.Delete(id)
	if err != nil && !sdk.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting dead host",
			fmt.Sprintf("Unable to delete dead host, unexpected error: %s", err.Error()),
		)
	}
}

func (r *deadHostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *deadHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importNumericID(ctx, req, resp)
}
