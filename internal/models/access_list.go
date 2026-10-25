// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strconv"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AccessListItemModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type AccessListClientModel struct {
	Address   types.String `tfsdk:"address"`
	Directive types.String `tfsdk:"directive"`
}

type AccessListModel struct {
	ID          types.String            `tfsdk:"id"`
	Name        types.String            `tfsdk:"name"`
	SatisfyAny  types.Bool              `tfsdk:"satisfy_any"`
	PassAuth    types.Bool              `tfsdk:"pass_auth"`
	Items       []AccessListItemModel   `tfsdk:"item"`
	Clients     []AccessListClientModel `tfsdk:"client"`
	OwnerUserID types.Int64             `tfsdk:"owner_user_id"`
}

// ToAPI returns the full access list to send to NPM. NPM only updates
// satisfy_any and pass_auth when the name is sent too, and replaces the
// items and clients with whatever is sent, so everything is always sent.
func (m AccessListModel) ToAPI() sdk.AccessListBase {
	accessList := sdk.AccessListBase{
		Name:       m.Name.ValueStringPointer(),
		SatisfyAny: m.SatisfyAny.ValueBoolPointer(),
		PassAuth:   m.PassAuth.ValueBoolPointer(),
		Items:      returnPointer([]sdk.AccessListItem{}),
		Clients:    returnPointer([]sdk.AccessListClient{}),
	}
	for _, item := range m.Items {
		*accessList.Items = append(*accessList.Items, sdk.AccessListItem{
			Username: item.Username.ValueStringPointer(),
			Password: item.Password.ValueStringPointer(),
		})
	}
	for _, client := range m.Clients {
		*accessList.Clients = append(*accessList.Clients, sdk.AccessListClient{
			Address:   client.Address.ValueStringPointer(),
			Directive: client.Directive.ValueStringPointer(),
		})
	}
	return accessList
}

// FromAPI sets the model from an access list returned by NPM. NPM doesn't
// return the passwords of the items, so they're taken from the items the
// model already has with the same username. Items NPM has that the model
// doesn't, e.g after an import, get an empty password, so the configured one
// is sent on the next apply.
func (m *AccessListModel) FromAPI(accessList sdk.AccessList) {
	m.ID = types.StringValue(strconv.Itoa(accessList.ID))
	m.Name = types.StringPointerValue(accessList.Name)
	m.SatisfyAny = boolValueOrFalse(accessList.SatisfyAny)
	m.PassAuth = boolValueOrFalse(accessList.PassAuth)
	m.OwnerUserID = types.Int64Value(int64(accessList.OwnerUserID))

	passwords := map[string]types.String{}
	for _, item := range m.Items {
		passwords[item.Username.ValueString()] = item.Password
	}
	m.Items = []AccessListItemModel{}
	for _, item := range derefSlice(accessList.Items) {
		if item.Username == nil {
			continue
		}
		password, ok := passwords[*item.Username]
		if !ok || password.IsNull() || password.IsUnknown() {
			password = types.StringValue("")
		}
		m.Items = append(m.Items, AccessListItemModel{
			Username: types.StringValue(*item.Username),
			Password: password,
		})
	}

	m.Clients = []AccessListClientModel{}
	for _, client := range derefSlice(accessList.Clients) {
		m.Clients = append(m.Clients, AccessListClientModel{
			Address:   types.StringPointerValue(client.Address),
			Directive: types.StringPointerValue(client.Directive),
		})
	}
}
