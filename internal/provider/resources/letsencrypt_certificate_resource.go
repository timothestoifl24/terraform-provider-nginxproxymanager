// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	utils "github.com/edklesel/terraform-provider-nginxproxymanager/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &letsencryptCertificateResource{}
	_ resource.ResourceWithConfigure   = &letsencryptCertificateResource{}
	_ resource.ResourceWithImportState = &letsencryptCertificateResource{}
	_ resource.ResourceWithModifyPlan  = &letsencryptCertificateResource{}
)

type letsencryptCertificateResource struct {
	client *sdk.Client
}

func NewLetsencryptCertificateResource() resource.Resource {
	return &letsencryptCertificateResource{}
}

func (r *letsencryptCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_letsencrypt_certificate"
}

// NPM doesn't return the DNS provider credentials, so they're unknown after
// an import. Setting them in the config then must not replace the
// certificate, only a change to credentials that were already known.
var requiresReplaceIfKnown = stringplanmodifier.RequiresReplaceIf(
	func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = !req.StateValue.IsNull()
	},
	"Changing the value replaces the certificate, unless it wasn't known yet, e.g after an import.",
	"Changing the value replaces the certificate, unless it wasn't known yet, e.g after an import.",
)

func (r *letsencryptCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_names": schema.SetAttribute{
				Required:    true,
				Description: "Domain names for the certificate.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"dns_challenge": schema.BoolAttribute{
				Required:    true,
				Description: "Whether or not to use a DNS challenge for this Letsencrypt certificate.",
				Validators: []validator.Bool{
					utils.RequireAttributesIfTrueValidator{
						RequiredAttrs: []string{"dns_provider", "dns_provider_credentials"},
					},
				},
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"dns_provider": schema.StringAttribute{
				Optional: true,
				Description: "Which DNS provider to use for the DNS challenge, as listed by the `nginxproxymanager_dns_providers` " +
					"data source. Must be set if `dns_challenge` is `true`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_provider_credentials": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Credentials for the provided DNS provider. Consult the Nginx Proxy Manager UI for the format for your provider. NPM doesn't return these, so they aren't checked for changes made outside Terraform.",
				PlanModifiers: []planmodifier.String{
					requiresReplaceIfKnown,
				},
			},
			"letsencrypt_email": schema.StringAttribute{
				Optional:           true,
				Description:        "Not used. Since NPM 2.13 the email address of the user that owns the certificate is used.",
				DeprecationMessage: "NPM ignores this since version 2.13 and uses the email address of the user that owns the certificate. Remove it from your configuration.",
			},
			"propagation_seconds": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of seconds to wait for the DNS challenge to propagate. Defaults to the provider default if left empty.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"nice_name": schema.StringAttribute{
				Computed:    true,
				Description: "Nice name for the certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ModifyPlan checks the DNS provider against the ones the running NPM
// supports, as that list changes between NPM versions.
func (r *letsencryptCertificateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil || req.Plan.Raw.IsNull() {
		return
	}

	var dnsProvider types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("dns_provider"), &dnsProvider)...)
	if resp.Diagnostics.HasError() || dnsProvider.IsNull() || dnsProvider.IsUnknown() {
		return
	}

	if !req.State.Raw.IsNull() {
		var current types.String
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("dns_provider"), &current)...)
		if current.Equal(dnsProvider) {
			return
		}
	}

	providers, err := r.client.Certificates.DNSProviders()
	if err != nil {
		resp.Diagnostics.AddWarning("Unable to check the DNS provider", err.Error())
		return
	}
	ids := make([]string, 0, len(providers))
	for _, p := range providers {
		ids = append(ids, p.ID)
	}
	if !slices.Contains(ids, dnsProvider.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("dns_provider"),
			"Unsupported DNS provider",
			fmt.Sprintf("NPM doesn't support the DNS provider %q. Supported providers are: %s", dnsProvider.ValueString(), strings.Join(ids, ", ")),
		)
	}
}

func (r *letsencryptCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.LetsencryptCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _certificate = sdk.CertificateBase{
		DomainNames: utils.ReturnPointer(models.StringValues(plan.DomainNames)),
		Provider:    utils.ReturnPointer("letsencrypt"),
		Meta: &sdk.CertificateMeta{
			DNSChallenge: plan.DNSChallenge.ValueBoolPointer(),
		},
	}

	if plan.DNSChallenge.ValueBool() {
		_certificate.Meta.DNSProvider = plan.DNSProvider.ValueStringPointer()
		_certificate.Meta.DNSProviderCredentials = plan.DNSProviderCredentials.ValueStringPointer()
	}

	if !plan.PropagationSeconds.IsNull() && !plan.PropagationSeconds.IsUnknown() {
		_certificate.Meta.PropagationSeconds = utils.ReturnPointer(int(plan.PropagationSeconds.ValueInt64()))
	}

	created, err := r.client.Certificates.Create(_certificate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating certificate.",
			fmt.Sprintf("Unable to create certificate, unexpected error: %s", err.Error()),
		)
		return
	}

	// Read it back, as the create response doesn't include everything
	certificate, err := r.client.Certificates.Get(created.ID)
	if err != nil {
		certificate = created
		resp.Diagnostics.AddWarning("Unable to read the certificate after creating it.", err.Error())
	}

	plan.FromAPI(certificate)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *letsencryptCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.LetsencryptCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	certificate, err := r.client.Certificates.Get(id)
	if err != nil {
		handleReadError(ctx, resp, "certificate", state.ID, err)
		return
	}

	state.FromAPI(certificate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Certificates can't be changed in NPM, so everything that matters requires
// replacement. What's left to update are the values NPM doesn't return,
// which are only kept in state.
func (r *letsencryptCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.LetsencryptCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *letsencryptCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.LetsencryptCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, ok := parseID(state.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	err := r.client.Certificates.Delete(id)
	if err != nil && !sdk.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting certificate",
			fmt.Sprintf("Unable to delete certificate, unexpected error: %s", err.Error()),
		)
	}
}

func (r *letsencryptCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *letsencryptCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importNumericID(ctx, req, resp)
}
