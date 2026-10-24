// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ resource.Resource                = &streamResource{}
	_ resource.ResourceWithConfigure   = &streamResource{}
	_ resource.ResourceWithImportState = &streamResource{}
)

type streamResource struct {
	client *sdk.Client
}

func NewStreamResource() resource.Resource {
	return &streamResource{}
}

func (r *streamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stream"
}

func (r *streamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the stream.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forwarding_host": schema.StringAttribute{
				Required:    true,
				Description: "Host to forward the stream to.",
			},
			"forwarding_port": schema.Int64Attribute{
				Required:    true,
				Description: "Port to forward the stream to.",
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"incoming_port": schema.Int64Attribute{
				Required:    true,
				Description: "Port the request is coming in on.",
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"tcp_forwarding": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable TCP Forwarding",
				Default:     booldefault.StaticBool(true),
			},
			"udp_forwarding": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable UDP Forwarding",
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable the stream.",
				Default:     booldefault.StaticBool(true),
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user which owns the stream.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *streamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.StreamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.Streams.Create(plan.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating stream.",
			fmt.Sprintf("Unable to create stream, unexpected error: %s", err.Error()),
		)
		return
	}

	// From here on the stream exists, so it's always saved to state, even if
	// the remaining steps fail. Terraform then marks it as tainted.
	var state models.StreamModel
	state.FromAPI(created)
	defer func() { resp.Diagnostics.Append(resp.State.Set(ctx, &state)...) }()

	if !applyEnabled(r.client.Streams, created.ID, true, plan.Enabled, "stream", &resp.Diagnostics) {
		return
	}

	obj, err := r.client.Streams.Get(created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading stream after creating it.", err.Error())
		return
	}
	state.FromAPI(obj)
	warnIfNginxOfflineMeta(obj.Meta, "stream", &resp.Diagnostics)
}

func (r *streamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.StreamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	obj, err := r.client.Streams.Get(id)
	if err != nil {
		handleReadError(ctx, resp, "stream", state.ID, err)
		return
	}

	state.FromAPI(obj)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *streamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.StreamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(plan.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	updated, err := r.client.Streams.Update(id, plan.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating stream.",
			fmt.Sprintf("Unable to update stream %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	enabled := updated.Enabled != nil && *updated.Enabled
	if !applyEnabled(r.client.Streams, id, enabled, plan.Enabled, "stream", &resp.Diagnostics) {
		return
	}

	obj, err := r.client.Streams.Get(id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading stream after updating it.", err.Error())
		return
	}

	var state models.StreamModel
	state.FromAPI(obj)
	warnIfNginxOfflineMeta(obj.Meta, "stream", &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *streamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.StreamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	err := r.client.Streams.Delete(id)
	if err != nil && !sdk.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting stream",
			fmt.Sprintf("Unable to delete stream, unexpected error: %s", err.Error()),
		)
	}
}

func (r *streamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *streamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importNumericID(ctx, req, resp)
}
