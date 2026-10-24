// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
)

func configureProvider(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (client *sdk.Client) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected configuration type",
			fmt.Sprintf("Expected *sdk.Client, got: %T. Please report this issue to the provider developer.", req.ProviderData),
		)
		return
	}

	return client
}

// parseID converts a resource ID from state to the integer NPM uses.
func parseID(id types.String, diags *diag.Diagnostics) (int, bool) {
	_id, err := strconv.Atoi(id.ValueString())
	if err != nil {
		diags.AddError(
			fmt.Sprintf("Unable to convert ID value %s to int type", id),
			err.Error(),
		)
		return 0, false
	}
	return _id, true
}

// handleReadError removes the resource from state if it no longer exists in
// NPM, so Terraform plans to create it again, and reports any other error.
func handleReadError(ctx context.Context, resp *resource.ReadResponse, kind string, id types.String, err error) {
	if sdk.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.AddError(
		fmt.Sprintf("Unable to read %s with ID %s.", kind, id.ValueString()),
		err.Error(),
	)
}

// importNumericID imports a resource by its NPM ID, rejecting anything that
// isn't a number with a clear message instead of failing on the next read.
func importNumericID(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected the numeric ID shown in the NPM UI or API, got %q.", req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type enabler interface {
	SetEnabled(id int, enabled bool) error
}

// applyEnabled enables or disables a host when the planned value differs from
// what NPM currently has.
func applyEnabled(client enabler, id int, current bool, planned types.Bool, kind string, diags *diag.Diagnostics) bool {
	if planned.IsNull() || planned.IsUnknown() || planned.ValueBool() == current {
		return true
	}
	if err := client.SetEnabled(id, planned.ValueBool()); err != nil {
		diags.AddError(fmt.Sprintf("Error enabling or disabling %s.", kind), err.Error())
		return false
	}
	return true
}

// warnIfNginxOffline reports a warning when NPM saved the host but nginx
// rejected the generated config. NPM keeps the host in that case, so it's not
// an error, but the host won't work until the config is fixed.
func warnIfNginxOffline(online types.Bool, nginxErr types.String, kind string, diags *diag.Diagnostics) {
	if online.IsNull() || online.ValueBool() {
		return
	}
	diags.AddWarning(
		fmt.Sprintf("Nginx rejected the config for this %s", kind),
		fmt.Sprintf("NPM saved the %s, but nginx reported an error loading it, so it is offline: %s", kind, nginxErr.ValueString()),
	)
}

// validateSSLSettings rejects SSL combinations NPM would silently change.
// NPM turns off ssl_forced and http2_support without a certificate, HSTS
// without ssl_forced and HSTS subdomains without HSTS, which would otherwise
// show up as an inconsistent result after apply.
func validateSSLSettings(certificateID types.Int64, sslForced, http2Support, hstsEnabled, hstsSubdomains types.Bool, diags *diag.Diagnostics) {
	noCertificate := certificateID.IsNull() || (!certificateID.IsUnknown() && certificateID.ValueInt64() == 0)

	if noCertificate {
		if sslForced.ValueBool() {
			diags.AddAttributeError(
				path.Root("ssl_forced"),
				"ssl_forced cannot be true when no certificate_id is given",
				"Forcing SSL requires a SSL Certificate to be set on the host",
			)
		}
		if http2Support.ValueBool() {
			diags.AddAttributeError(
				path.Root("http2_support"),
				"http2_support cannot be true when no certificate_id is given",
				"Enabling HTTP2 support requires a Certificate to be set",
			)
		}
	}

	// hsts_enabled depends on ssl_forced being true
	if !sslForced.IsUnknown() && !sslForced.ValueBool() && hstsEnabled.ValueBool() {
		diags.AddAttributeError(
			path.Root("hsts_enabled"),
			"hsts_enabled cannot be true when ssl_forced is false",
			"Enabling HSTS requires SSL to be forced",
		)
	}
	// hsts_subdomains depends on hsts_enabled being true
	if !hstsEnabled.IsUnknown() && !hstsEnabled.ValueBool() && hstsSubdomains.ValueBool() {
		diags.AddAttributeError(
			path.Root("hsts_subdomains"),
			"hsts_subdomains cannot be true when hsts_enabled is false",
			"Enabling HSTS for Subdomains requires HSTS to be enabled",
		)
	}
}

// warnIfNginxOfflineMeta is warnIfNginxOffline for hosts that don't expose
// nginx_online as an attribute.
func warnIfNginxOfflineMeta(meta *sdk.HostMeta, kind string, diags *diag.Diagnostics) {
	if meta == nil || meta.NginxOnline == nil {
		return
	}
	nginxErr := types.StringNull()
	if meta.NginxErr != nil {
		nginxErr = types.StringValue(*meta.NginxErr)
	}
	warnIfNginxOffline(types.BoolValue(*meta.NginxOnline), nginxErr, kind, diags)
}
