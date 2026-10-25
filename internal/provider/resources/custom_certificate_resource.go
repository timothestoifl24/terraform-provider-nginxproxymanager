// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	utils "github.com/edklesel/terraform-provider-nginxproxymanager/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var (
	_ resource.Resource                = &customCertificateResource{}
	_ resource.ResourceWithConfigure   = &customCertificateResource{}
	_ resource.ResourceWithImportState = &customCertificateResource{}
)

type customCertificateResource struct {
	client *sdk.Client
}

func NewCustomCertificateResource() resource.Resource {
	return &customCertificateResource{}
}

func (r *customCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_certificate"
}

func (r *customCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nice_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Nice name for the certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"certificate": schema.StringAttribute{
				Required:    true,
				Description: "Body of the certificate in PEM format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Private key of the certificate in PEM format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"intermediate_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "Intermediate certificate in the chain. NPM doesn't return it, so changes made outside Terraform aren't detected.",
				PlanModifiers: []planmodifier.String{
					requiresReplaceIfKnown,
				},
			},
		},
	}
}

func (r *customCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.CustomCertificateModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _certificate = sdk.CertificateBase{
		Provider: utils.ReturnPointer("other"),
	}
	if !plan.NiceName.IsUnknown() {
		_certificate.NiceName = plan.NiceName.ValueStringPointer()
	}

	certUpload := sdk.CertificateUpload{
		Certificate:    plan.Certificate.ValueStringPointer(),
		CertificateKey: plan.CertificateKey.ValueStringPointer(),
	}
	if !plan.IntermediateCertificate.IsNull() {
		certUpload.IntermediateCertificate = plan.IntermediateCertificate.ValueStringPointer()
	}

	// First, validate the certificate
	err := r.client.Certificates.Validate(certUpload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error validating certificate.",
			fmt.Sprintf("Unable to validate certificate: %s", err.Error()),
		)
		return
	}

	// Then create the certificate object
	certificate, err := r.client.Certificates.Create(_certificate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating certificate.",
			fmt.Sprintf("Unable to create certificate, unexpected error: %s", err.Error()),
		)
		return
	}

	// Then, upload the certificate body
	_, err = r.client.Certificates.Upload(certificate.ID, certUpload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error uploading certificate.",
			fmt.Sprintf("Unable to upload certificate body, unexpected error: %s", err.Error()),
		)

		err = r.client.Certificates.Delete(certificate.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error tidying up failed certificate.",
				fmt.Sprintf("Unable to tidy up certificate which failed to upload, unexpected error: %s", err.Error()),
			)
		}
		return
	}

	certGet, err := r.client.Certificates.Get(certificate.ID)
	if err != nil {
		certGet = certificate
		resp.Diagnostics.AddWarning("Unable to read the certificate after creating it.", err.Error())
	}
	plan.FromAPI(certGet)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.CustomCertificateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to convert ID value %s to int type", state.ID),
			err.Error(),
		)
		return
	}

	certificate, err := r.client.Certificates.Get(id)
	if err != nil {
		handleReadError(ctx, resp, "certificate", state.ID, err)
		return
	}

	state.FromAPI(certificate)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Certificates can't be changed in NPM, so everything that matters requires
// replacement. What's left to update are the values NPM doesn't return,
// which are only kept in state.
func (r *customCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.CustomCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *customCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state models.CustomCertificateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to convert ID value %s to int type", state.ID),
			err.Error(),
		)
		return
	}

	err = r.client.Certificates.Delete(id)
	if err != nil && !sdk.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting certificate",
			fmt.Sprintf("Unable to delete certificate, unexpected error: %s", err.Error()),
		)
	}
}

func (r *customCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *customCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importNumericID(ctx, req, resp)
}
