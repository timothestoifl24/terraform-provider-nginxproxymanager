// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func returnPointer[S any](s S) *S {
	return &s
}

func derefSlice[S any](s *[]S) []S {
	if s == nil {
		return []S{}
	}
	return *s
}

// StringValues converts a list of Terraform strings to Go strings.
func StringValues(values []types.String) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v.ValueString())
	}
	return out
}

// StringsValue converts a list of Go strings to Terraform strings.
func StringsValue(values []string) []types.String {
	out := make([]types.String, 0, len(values))
	for _, v := range values {
		out = append(out, types.StringValue(v))
	}
	return out
}

// NPM leaves out fields it has no value for, e.g on hosts created by older
// versions, so missing values are read as the defaults NPM itself uses.

func intPointerValue(i *int) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*i))
}

func intValueOrZero(i *int) types.Int64 {
	if i == nil {
		return types.Int64Value(0)
	}
	return types.Int64Value(int64(*i))
}

func boolValueOrFalse(b *bool) types.Bool {
	if b == nil {
		return types.BoolValue(false)
	}
	return types.BoolValue(*b)
}

func stringValueOrEmpty(s *string) types.String {
	if s == nil {
		return types.StringValue("")
	}
	return types.StringValue(*s)
}

// NPM adds the result of the last nginx reload to the meta of every host.
func hostMetaValues(meta *sdk.HostMeta) (online types.Bool, nginxErr types.String) {
	online, nginxErr = types.BoolNull(), types.StringNull()
	if meta == nil {
		return
	}
	if meta.NginxOnline != nil {
		online = types.BoolValue(*meta.NginxOnline)
	}
	if meta.NginxErr != nil {
		nginxErr = types.StringValue(*meta.NginxErr)
	}
	return
}
