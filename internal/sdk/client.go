// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package go_npm_proxy_manager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	version "github.com/hashicorp/go-version"
)

// Tokens are refreshed when they are this close to expiring.
const tokenRefreshWindow = 5 * time.Minute

type Client struct {
	HostURL string     `json:"hostUrl"`
	Auth    AuthStruct `json:"auth"`
	Token   string
	Version *version.Version

	// When the current token expires. Zero means unknown, in which case
	// the token is only renewed after NPM rejects it.
	tokenExpires time.Time
	tokenMu      sync.Mutex
	httpClient   *http.Client

	Users            *crudObject[User, UserBase, UserCreate]
	ProxyHosts       *crudObject[ProxyHost, ProxyHostBase, ProxyHostBase]
	RedirectionHosts *crudObject[RedirectionHost, RedirectionHostBase, RedirectionHostBase]
	DeadHosts        *crudObject[DeadHost, DeadHostBase, DeadHostBase]
	Streams          *crudObject[Stream, StreamBase, StreamBase]
	AccessLists      *crudObject[AccessList, AccessListBase, AccessListBase]
	Certificates     *certObject
	Settings         *settingsObject
}

type AuthStruct struct {
	Identity string `json:"identity"`
	Secret   string `json:"secret"`
}

type ClientOptions struct {
	// An existing API token. When set, the username and password are only
	// used to log in again if the token can no longer be refreshed.
	Token string
	// Timeout for a single request. Certificate requests can block for
	// several minutes while Letsencrypt validates the domains.
	Timeout time.Duration
}

type TokenResponseStruct struct {
	Token       string `json:"token"`
	Expires     string `json:"expires"`
	Requires2FA bool   `json:"requires_2fa"`
}

type HTTPResponseError struct {
	Error HTTPResponseCode `json:"error"`
}

type HTTPResponseCode struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// APIError is returned for any unsuccessful response from NPM.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("error received from NPM API (HTTP %d): %s", e.StatusCode, e.Message)
}

// IsNotFound reports whether err means the object doesn't exist in NPM.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// isAuthError reports whether NPM rejected the token. NPM answers 400, 401
// or 403 depending on where the token check fails, so the message is
// checked as well as the status code.
func isAuthError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode == http.StatusUnauthorized {
		return true
	}
	msg := strings.ToLower(apiErr.Message)
	for _, m := range []string{
		"token has expired",
		"empty token",
		"user cannot be loaded for token",
		// Token errors raised inside a permission check are reported as this
		"permission denied",
	} {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}

type ClientVersion struct {
	Major    int64 `json:"major"`
	Minor    int64 `json:"minor"`
	Revision int64 `json:"revision"`
}

type ClientAPIRoot struct {
	Status  string        `json:"status"`
	Setup   bool          `json:"setup"`
	Version ClientVersion `json:"version"`
}

func CheckResponse(response []byte) *HTTPResponseError {
	var httperr HTTPResponseError

	// Check for boolean responses, which won't unmarshal properly
	if slices.Contains([]string{"true", "false"}, string(response)) {
		return nil
	}

	// If the response is not json, return a generic error code and the response
	if !json.Valid(response) {
		httperr = HTTPResponseError{
			Error: HTTPResponseCode{
				Code:    999,
				Message: string(response),
			},
		}
		return &httperr
	}

	// Otherwise if the response unmarshals successfully, then the
	// response is a json object (although it might not be an error)
	err := json.Unmarshal(response, &httperr)
	if err == nil {
		// If the Error.Code is 0, then the json returned wasn't an error and
		// we can return no error
		if httperr.Error.Code == 0 && httperr.Error.Message == "" {
			return nil
		}
		// Otherwise, the response will be an error
		return &httperr

		// Lastly, if the json couldn't unmarshal properly, then the json body was
		// likely a list of items (i.e begins with "["), in which case there is
		// also no error to return
	} else {
		return nil
	}

}

// NewClient creates a client and panics if NPM can't be reached. The
// provider uses NewClientWithOptions instead, which returns the error.
func NewClient(url string, auth AuthStruct) *Client {
	client, err := NewClientWithOptions(url, auth, ClientOptions{})
	if err != nil {
		panic(err.Error())
	}
	return client
}

func NewClientWithOptions(url string, auth AuthStruct, opts ClientOptions) (*Client, error) {

	if opts.Timeout == 0 {
		opts.Timeout = 20 * time.Minute
	}

	var client = &Client{
		HostURL: strings.TrimRight(url, "/"),
		Auth:    auth,
		Token:   opts.Token,
	}

	_client := retryablehttp.NewClient()
	_client.Logger = nil
	_client.RetryMax = 3
	_client.CheckRetry = checkRetry
	_client.HTTPClient.Timeout = opts.Timeout
	client.httpClient = _client.StandardClient()

	client.Users = &crudObject[User, UserBase, UserCreate]{
		client:          client,
		apiPath:         "users",
		expand:          "permissions",
		writeOnlyFields: []string{},
	}
	client.ProxyHosts = &crudObject[ProxyHost, ProxyHostBase, ProxyHostBase]{
		client:          client,
		apiPath:         "nginx/proxy-hosts",
		writeOnlyFields: []string{},
	}
	client.RedirectionHosts = &crudObject[RedirectionHost, RedirectionHostBase, RedirectionHostBase]{
		client:          client,
		apiPath:         "nginx/redirection-hosts",
		writeOnlyFields: []string{},
	}
	client.DeadHosts = &crudObject[DeadHost, DeadHostBase, DeadHostBase]{
		client:          client,
		apiPath:         "nginx/dead-hosts",
		writeOnlyFields: []string{},
	}
	client.Streams = &crudObject[Stream, StreamBase, StreamBase]{
		client:          client,
		apiPath:         "nginx/streams",
		writeOnlyFields: []string{},
	}
	client.AccessLists = &crudObject[AccessList, AccessListBase, AccessListBase]{
		client:          client,
		apiPath:         "nginx/access-lists",
		expand:          "items,clients",
		writeOnlyFields: []string{},
	}
	client.Certificates = &certObject{
		client:          client,
		apiPath:         "nginx/certificates",
		writeOnlyFields: []string{"Meta.DNSProviderCredentials"},
	}
	client.Settings = &settingsObject{
		client:  client,
		apiPath: "settings",
	}

	_ver, err := client.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve NPM version from API: %s", err)
	}
	client.Version = _ver

	return client, nil
}

// Only retry when the request most likely never reached NPM. Retrying a
// POST that failed with a 500 could otherwise create the object twice.
func checkRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	switch resp.StatusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true, nil
	}
	return false, nil
}

// GetToken logs in with the configured credentials.
func (c *Client) GetToken() error {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	return c.login()
}

func (c *Client) login() error {

	if c.Auth.Identity == "" || c.Auth.Secret == "" {
		return fmt.Errorf("no username and password configured to log in with")
	}

	// Convert the username/password to json
	body, err := json.Marshal(c.Auth)
	if err != nil {
		return fmt.Errorf("error encoding client credentials to json: %s", err)
	}

	// Make the request for the token and parse the json response for the token value
	_response, err := c.doRequest("tokens", http.MethodPost, body, "", "application/json")
	if err != nil {
		return fmt.Errorf("unable to get client token: %s", err.Error())
	}

	return c.setToken(_response)
}

// refresh swaps the current token for a new one using GET /api/tokens,
// which only works while the current token is still valid.
func (c *Client) refresh() error {
	_response, err := c.doRequest("tokens", http.MethodGet, nil, c.Token, "")
	if err != nil {
		return err
	}
	return c.setToken(_response)
}

func (c *Client) setToken(_response []byte) error {
	var response TokenResponseStruct
	if err := json.Unmarshal(_response, &response); err != nil {
		return fmt.Errorf("unable to parse token response: %s", err)
	}
	if response.Requires2FA {
		return fmt.Errorf("the user has two-factor authentication enabled, which the provider can't complete. " +
			"Configure the provider with a token instead, e.g. a long-lived one from GET /api/tokens?expiry=1y")
	}
	if response.Token == "" {
		return fmt.Errorf("no token returned by NPM")
	}
	c.Token = response.Token
	c.tokenExpires = time.Time{}
	if expires, err := time.Parse(time.RFC3339, response.Expires); err == nil {
		c.tokenExpires = expires
	}
	return nil
}

// ensureToken returns the token to use for a request. It logs in if there is
// no token yet, refreshes one that is about to expire, and logs in again when
// force is set because NPM rejected the current token.
func (c *Client) ensureToken(force bool) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.Token == "" {
		if err := c.login(); err != nil {
			return "", err
		}
		return c.Token, nil
	}

	expiring := !c.tokenExpires.IsZero() && time.Until(c.tokenExpires) < tokenRefreshWindow
	if !force && !expiring {
		return c.Token, nil
	}

	// A token that is still valid can be swapped for a new one. One that was
	// rejected needs a fresh login.
	if !force {
		if err := c.refresh(); err == nil {
			return c.Token, nil
		}
	}
	if err := c.login(); err != nil {
		return "", err
	}
	return c.Token, nil
}

func (c *Client) NewRequest(endpoint string, method string, body *[]byte, auth bool, content_type *string) (response []byte, err error) {

	var _body []byte
	var _contentType string
	if body != nil {
		if content_type == nil {
			return nil, fmt.Errorf("Content-Type is required if a request body is provided")
		}
		_body = *body
		_contentType = *content_type
	}

	if !auth {
		return c.doRequest(endpoint, method, _body, "", _contentType)
	}

	token, err := c.ensureToken(false)
	if err != nil {
		return nil, err
	}
	response, err = c.doRequest(endpoint, method, _body, token, _contentType)

	// If the token was rejected, log in again and retry once
	if err != nil && isAuthError(err) && c.Auth.Identity != "" {
		token, loginErr := c.ensureToken(true)
		if loginErr != nil {
			return nil, fmt.Errorf("%s (logging in again also failed: %s)", err, loginErr)
		}
		response, err = c.doRequest(endpoint, method, _body, token, _contentType)
	}

	return response, err
}

func (c *Client) doRequest(endpoint string, method string, body []byte, token string, contentType string) (response []byte, err error) {

	// Format the URL based on the endpoint passed in
	url := fmt.Sprintf("%s/api/%s", c.HostURL, endpoint)

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to %s: %s", req.URL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response
	response, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error unwrapping response from %s: %s", req.URL, err)
	}

	// Check if the response is an error
	if httpErr := CheckResponse(response); httpErr != nil {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: httpErr.Error.Message}
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(response))}
	}

	return response, nil
}

func (c *Client) GetVersion() (ver *version.Version, err error) {

	var response ClientAPIRoot
	_response, err := c.NewRequest("", http.MethodGet, nil, false, nil)
	if err != nil {
		return ver, fmt.Errorf("unable to get version of running NPM: %v", err)
	}

	err = json.Unmarshal(_response, &response)
	if err != nil {
		return
	}

	_ver, err := version.NewVersion(fmt.Sprintf("%d.%d.%d", response.Version.Major, response.Version.Minor, response.Version.Revision))
	if err != nil {
		return nil, err
	}

	return _ver, nil
}
