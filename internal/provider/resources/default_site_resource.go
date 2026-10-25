// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	utils "github.com/edklesel/terraform-provider-nginxproxymanager/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultSiteID = "default-site"

var (
	_ resource.Resource                   = &defaultSiteResource{}
	_ resource.ResourceWithConfigure      = &defaultSiteResource{}
	_ resource.ResourceWithImportState    = &defaultSiteResource{}
	_ resource.ResourceWithValidateConfig = &defaultSiteResource{}
)

type defaultSiteModel struct {
	ID       types.String `tfsdk:"id"`
	Page     types.String `tfsdk:"page"`
	Redirect types.String `tfsdk:"redirect"`
	HTML     types.String `tfsdk:"html"`
}

type defaultSiteResource struct {
	client *sdk.Client
}

func NewDefaultSiteResource() resource.Resource {
	return &defaultSiteResource{}
}

func (r *defaultSiteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_default_site"
}

func (r *defaultSiteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "What NPM shows when nginx is hit with a host it doesn't know. There is only one of these, " +
			"so only declare it once. Destroying it resets NPM to the congratulations page.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Always `default-site`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"page": schema.StringAttribute{
				Required: true,
				Description: "What to show: `congratulations` (the NPM default page), `404` (a not found page), " +
					"`444` (close the connection), `redirect` (redirect to `redirect`) or `html` (show `html`).",
				Validators: []validator.String{
					stringvalidator.OneOf("congratulations", "404", "444", "redirect", "html"),
				},
			},
			"redirect": schema.StringAttribute{
				Optional:    true,
				Description: "URL to redirect to, when `page` is `redirect`.",
			},
			"html": schema.StringAttribute{
				Optional:    true,
				Description: "HTML to show, when `page` is `html`.",
			},
		},
	}
}

func (r *defaultSiteResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data defaultSiteModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Page.IsUnknown() {
		return
	}

	for attr, value := range map[string]types.String{"redirect": data.Redirect, "html": data.HTML} {
		if data.Page.ValueString() == attr && value.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root(attr),
				fmt.Sprintf("%s is required", attr),
				fmt.Sprintf("%s must be set when page is %q.", attr, attr),
			)
		}
		if data.Page.ValueString() != attr && !value.IsNull() && !value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root(attr),
				fmt.Sprintf("%s is only used when page is %q", attr, attr),
				fmt.Sprintf("NPM ignores %s unless page is %q, remove it or change page.", attr, attr),
			)
		}
	}
}

func (m defaultSiteModel) toAPI() sdk.SettingBase {
	setting := sdk.SettingBase{
		Value: m.Page.ValueStringPointer(),
		Meta:  &sdk.SettingMeta{},
	}
	switch m.Page.ValueString() {
	case "redirect":
		setting.Meta.Redirect = m.Redirect.ValueStringPointer()
	case "html":
		setting.Meta.HTML = m.HTML.ValueStringPointer()
	}
	return setting
}

func (m *defaultSiteModel) fromAPI(setting sdk.Setting) {
	m.ID = types.StringValue(setting.ID)
	m.Page = types.StringPointerValue(setting.Value)
	m.Redirect, m.HTML = types.StringNull(), types.StringNull()
	if setting.Meta == nil {
		return
	}
	// NPM keeps the meta of the other pages, so only the one in use is read
	switch m.Page.ValueString() {
	case "redirect":
		m.Redirect = types.StringPointerValue(setting.Meta.Redirect)
	case "html":
		m.HTML = types.StringPointerValue(setting.Meta.HTML)
	}
}

func (r *defaultSiteResource) apply(plan defaultSiteModel, diags *diag.Diagnostics) (defaultSiteModel, bool) {
	setting, err := r.client.Settings.Update(defaultSiteID, plan.toAPI())
	if err != nil {
		diags.AddError("Error updating the default site.", err.Error())
		return plan, false
	}
	var state defaultSiteModel
	state.fromAPI(setting)
	return state, true
}

func (r *defaultSiteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan defaultSiteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state, ok := r.apply(plan, &resp.Diagnostics); ok {
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	}
}

func (r *defaultSiteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	setting, err := r.client.Settings.Get(defaultSiteID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the default site.", err.Error())
		return
	}

	var state defaultSiteModel
	state.fromAPI(setting)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *defaultSiteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan defaultSiteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state, ok := r.apply(plan, &resp.Diagnostics); ok {
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	}
}

// The setting can't be deleted, so it's reset to what NPM starts with.
func (r *defaultSiteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, err := r.client.Settings.Update(defaultSiteID, sdk.SettingBase{
		Value: utils.ReturnPointer("congratulations"),
		Meta:  &sdk.SettingMeta{},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error resetting the default site.", err.Error())
	}
}

func (r *defaultSiteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *defaultSiteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != defaultSiteID {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("The default site is imported with the ID %q, got %q.", defaultSiteID, req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
