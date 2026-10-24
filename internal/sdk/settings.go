// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package go_npm_proxy_manager

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type SettingMeta struct {
	Redirect *string `json:"redirect,omitempty"`
	HTML     *string `json:"html,omitempty"`
}

type SettingBase struct {
	Value *string      `json:"value,omitempty"`
	Meta  *SettingMeta `json:"meta,omitempty"`
}

type Setting struct {
	SettingBase
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Settings are identified by a string ID and can't be created or deleted,
// only read and updated.
type settingsObject struct {
	client  *Client
	apiPath string
}

func (o settingsObject) List() (settings []Setting, err error) {
	response, err := o.client.NewRequest(o.apiPath, http.MethodGet, nil, true, nil)
	if err != nil {
		return settings, fmt.Errorf("unable to list settings: %w", err)
	}
	if err = json.Unmarshal(response, &settings); err != nil {
		return settings, fmt.Errorf("unable to unmarshal json response: %s", err)
	}
	return settings, nil
}

func (o settingsObject) Get(id string) (setting Setting, err error) {
	response, err := o.client.NewRequest(fmt.Sprintf("%s/%s", o.apiPath, id), http.MethodGet, nil, true, nil)
	if err != nil {
		return setting, fmt.Errorf("unable to get setting %s: %w", id, err)
	}
	if err = json.Unmarshal(response, &setting); err != nil {
		return setting, fmt.Errorf("unable to unmarshal json response: %s", err)
	}
	return setting, nil
}

// Update a setting. NPM can answer with an empty body when reloading nginx
// fails, so the setting is read back rather than trusting the response.
func (o settingsObject) Update(id string, newProperties SettingBase) (setting Setting, err error) {
	body, err := json.Marshal(newProperties)
	if err != nil {
		return setting, fmt.Errorf("unable to marshal json body: %s", err)
	}

	_, err = o.client.NewRequest(
		fmt.Sprintf("%s/%s", o.apiPath, id),
		http.MethodPut,
		&body,
		true,
		returnPointer("application/json"),
	)
	if err != nil {
		return setting, fmt.Errorf("unable to update setting %s: %w", id, err)
	}

	return o.Get(id)
}
