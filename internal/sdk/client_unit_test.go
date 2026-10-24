// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package go_npm_proxy_manager

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeNPM is a minimal stand-in for the NPM API, for testing how the client
// handles tokens and errors without a running NPM.
type fakeNPM struct {
	logins    atomic.Int32
	refreshes atomic.Int32
	posts     atomic.Int32
	// The token NPM currently accepts.
	validToken atomic.Value
	// How long issued tokens are valid for.
	expiresIn time.Duration
}

func newFakeNPM(t *testing.T) (*fakeNPM, *httptest.Server) {
	f := &fakeNPM{expiresIn: 24 * time.Hour}
	f.validToken.Store("")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"OK","setup":true,"version":{"major":2,"minor":16,"revision":0}}`)
	})
	issue := func(w http.ResponseWriter) {
		token := fmt.Sprintf("token-%d", f.logins.Load()+f.refreshes.Load())
		f.validToken.Store(token)
		_, _ = fmt.Fprintf(w, `{"token":%q,"expires":%q}`, token, time.Now().Add(f.expiresIn).UTC().Format(time.RFC3339))
	}
	mux.HandleFunc("POST /api/tokens", func(w http.ResponseWriter, r *http.Request) {
		f.logins.Add(1)
		issue(w)
	})
	mux.HandleFunc("GET /api/tokens", func(w http.ResponseWriter, r *http.Request) {
		if !f.authorized(r) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"error":{"code":400,"message":"Token has expired"}}`)
			return
		}
		f.refreshes.Add(1)
		issue(w)
	})
	mux.HandleFunc("GET /api/nginx/proxy-hosts/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !f.authorized(r) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"error":{"code":400,"message":"Token has expired"}}`)
			return
		}
		if r.PathValue("id") != "1" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, `{"error":{"code":404,"message":"Not Found - %s"}}`, r.PathValue("id"))
			return
		}
		_, _ = fmt.Fprint(w, `{"id":1,"domain_names":["a.example"],"locations":null}`)
	})
	mux.HandleFunc("POST /api/nginx/proxy-hosts", func(w http.ResponseWriter, r *http.Request) {
		f.posts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":{"code":500,"message":"Internal Error"}}`)
	})
	mux.HandleFunc("POST /api/nginx/proxy-hosts/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":{"code":400,"message":"Host is already enabled"}}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return f, server
}

func (f *fakeNPM) authorized(r *http.Request) bool {
	token := f.validToken.Load().(string)
	return token != "" && r.Header.Get("Authorization") == "Bearer "+token
}

func newFakeClient(t *testing.T, url string, opts ClientOptions) *Client {
	client, err := NewClientWithOptions(url, AuthStruct{Identity: "admin@example.com", Secret: "changeme"}, opts)
	assert.NoError(t, err)
	return client
}

func TestUnitGetNotFound(t *testing.T) {
	_, server := newFakeNPM(t)
	client := newFakeClient(t, server.URL, ClientOptions{})

	host, err := client.ProxyHosts.Get(1)
	assert.NoError(t, err)
	assert.Equal(t, 1, host.ID)
	assert.NotNil(t, host.Locations, "null locations should be returned as an empty slice")

	_, err = client.ProxyHosts.Get(2)
	assert.Error(t, err)
	assert.True(t, IsNotFound(err), "expected a not found error, got %s", err)
}

func TestUnitLogsInAgainWhenTokenRejected(t *testing.T) {
	f, server := newFakeNPM(t)
	client := newFakeClient(t, server.URL, ClientOptions{})

	_, err := client.ProxyHosts.Get(1)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), f.logins.Load())

	// NPM no longer accepts the token, e.g because it expired
	f.validToken.Store("something-else")

	_, err = client.ProxyHosts.Get(1)
	assert.NoError(t, err)
	assert.Equal(t, int32(2), f.logins.Load())
}

func TestUnitRefreshesExpiringToken(t *testing.T) {
	f, server := newFakeNPM(t)
	f.expiresIn = time.Minute
	client := newFakeClient(t, server.URL, ClientOptions{})

	_, err := client.ProxyHosts.Get(1)
	assert.NoError(t, err)
	_, err = client.ProxyHosts.Get(1)
	assert.NoError(t, err)

	assert.Equal(t, int32(1), f.logins.Load())
	assert.Equal(t, int32(1), f.refreshes.Load())
}

func TestUnitConfiguredToken(t *testing.T) {
	f, server := newFakeNPM(t)
	f.validToken.Store("long-lived")
	client, err := NewClientWithOptions(server.URL, AuthStruct{}, ClientOptions{Token: "long-lived"})
	assert.NoError(t, err)

	_, err = client.ProxyHosts.Get(1)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), f.logins.Load())

	// Without credentials there's nothing to fall back to
	f.validToken.Store("something-else")
	_, err = client.ProxyHosts.Get(1)
	assert.ErrorContains(t, err, "Token has expired")
}

func TestUnitNoRetryOnServerError(t *testing.T) {
	f, server := newFakeNPM(t)
	client := newFakeClient(t, server.URL, ClientOptions{})

	_, err := client.ProxyHosts.Create(ProxyHostBase{})
	assert.Error(t, err)
	assert.Equal(t, int32(1), f.posts.Load(), "a failed create must not be retried")
}

func TestUnitEnableAlreadyEnabled(t *testing.T) {
	_, server := newFakeNPM(t)
	client := newFakeClient(t, server.URL, ClientOptions{})

	assert.NoError(t, client.ProxyHosts.SetEnabled(1, true))
}

func TestUnitTwoFactor(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"OK","setup":true,"version":{"major":2,"minor":16,"revision":0}}`)
	})
	mux.HandleFunc("POST /api/tokens", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"requires_2fa":true,"challenge_token":"abc"}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := newFakeClient(t, server.URL, ClientOptions{})
	err := client.GetToken()
	assert.ErrorContains(t, err, "two-factor")
}
