// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ resource.Resource                = &accessListResource{}
	_ resource.ResourceWithConfigure   = &accessListResource{}
	_ resource.ResourceWithImportState = &accessListResource{}
)

type accessListResource struct {
	client *sdk.Client
}

func NewAccessListResource() resource.Resource {
	return &accessListResource{}
}

func (r *accessListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_list"
}

func (r *accessListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the access list.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the access list.",
			},
			"satisfy_any": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether a request should pass if it satisfies any of the rules.",
			},
			"pass_auth": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the to pass auth to the host.",
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user which owns the access list.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			// NPM inserts the users in parallel, so it doesn't keep their order
			"item": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Description: "Basic auth username.",
							Required:    true,
						},
						"password": schema.StringAttribute{
							Description: "Basic auth password. NPM doesn't return passwords, so a password changed outside Terraform isn't detected.",
							Required:    true,
							Sensitive:   true,
						},
					},
				},
			},
			"client": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Description: "IP Address of a client.",
							Required:    true,
						},
						"directive": schema.StringAttribute{
							Description: "Directive, allow or deny.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("allow", "deny"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *accessListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.AccessListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.AccessLists.Create(plan.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating access list.",
			fmt.Sprintf("Unable to create access list, unexpected error: %s", err.Error()),
		)
		return
	}

	// The create response doesn't include the items and clients
	accessList, err := r.client.AccessLists.Get(created.ID)
	if err != nil {
		accessList = created
		resp.Diagnostics.AddWarning("Unable to read the access list after creating it.", err.Error())
	}

	plan.FromAPI(accessList)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *accessListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.AccessListModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	accessList, err := r.client.AccessLists.Get(id)
	if err != nil {
		handleReadError(ctx, resp, "access list", state.ID, err)
		return
	}

	state.FromAPI(accessList)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accessListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.AccessListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(plan.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	if _, err := r.client.AccessLists.Update(id, plan.ToAPI()); err != nil {
		resp.Diagnostics.AddError(
			"Error updating access list.",
			fmt.Sprintf("Unable to update access list %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	accessList, err := r.client.AccessLists.Get(id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading access list after updating it.", err.Error())
		return
	}

	plan.FromAPI(accessList)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *accessListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.AccessListModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	err := r.client.AccessLists.Delete(id)
	if err != nil && !sdk.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting access list",
			fmt.Sprintf("Unable to delete access list, unexpected error: %s", err.Error()),
		)
	}
}

func (r *accessListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *accessListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importNumericID(ctx, req, resp)
}
