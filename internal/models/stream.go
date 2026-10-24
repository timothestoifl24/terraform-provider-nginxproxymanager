// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strconv"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type StreamModel struct {
	ID             types.String `tfsdk:"id"`
	IncomingPort   types.Int64  `tfsdk:"incoming_port"`
	TCPForwarding  types.Bool   `tfsdk:"tcp_forwarding"`
	UDPForwarding  types.Bool   `tfsdk:"udp_forwarding"`
	ForwardingHost types.String `tfsdk:"forwarding_host"`
	ForwardingPort types.Int64  `tfsdk:"forwarding_port"`
	OwnerUserID    types.Int64  `tfsdk:"owner_user_id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
}

// ToAPI returns the fields to send to NPM when creating or updating the
// stream. Enabled is set through the enable/disable endpoints.
func (m StreamModel) ToAPI() sdk.StreamBase {
	return sdk.StreamBase{
		IncomingPort:   returnPointer(int(m.IncomingPort.ValueInt64())),
		ForwardingHost: m.ForwardingHost.ValueStringPointer(),
		ForwardingPort: returnPointer(int(m.ForwardingPort.ValueInt64())),
		TCPForwarding:  m.TCPForwarding.ValueBoolPointer(),
		UDPForwarding:  m.UDPForwarding.ValueBoolPointer(),
	}
}

// FromAPI sets the model from a stream returned by NPM.
func (m *StreamModel) FromAPI(stream sdk.Stream) {
	m.ID = types.StringValue(strconv.Itoa(stream.ID))
	m.OwnerUserID = types.Int64Value(int64(stream.OwnerUserID))
	m.IncomingPort = intPointerValue(stream.IncomingPort)
	m.ForwardingHost = types.StringPointerValue(stream.ForwardingHost)
	m.ForwardingPort = intPointerValue(stream.ForwardingPort)
	m.TCPForwarding = boolValueOrFalse(stream.TCPForwarding)
	m.UDPForwarding = boolValueOrFalse(stream.UDPForwarding)
	m.Enabled = boolValueOrFalse(stream.Enabled)
}
