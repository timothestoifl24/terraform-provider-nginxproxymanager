// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Domains in an order NPM doesn't keep, SSL settings, locations and a
// disabled host all have to round-trip without a diff.
func TestAccProxyHostResourceRoundTrip(t *testing.T) {

	var id = rand.IntN(1e4)
	const name = "nginxproxymanager_proxy_host.test"
	certConfig := customCertificateConfig(fmt.Sprintf("cert_%d", id))

	config := func(enabled bool, sslForced bool) string {
		return providerConfig + certConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host   = "10.10.0.1"
  forward_port   = 8080
  forward_scheme = "http"
  domain_names = [
    "zz_%[1]d.example.com",
    "aa_%[1]d.example.com",
    "mm_%[1]d.example.com",
  ]

  certificate_id        = nginxproxymanager_custom_certificate.cert_%[1]d.id
  ssl_forced            = %[3]t
  hsts_enabled          = %[3]t
  http2_support         = true
  trust_forwarded_proto = true
  enabled               = %[2]t

  location {
    path            = "/api"
    forward_scheme  = "http"
    forward_host    = "10.10.0.2"
    forward_port    = 9000
    forward_path    = "/v1"
    advanced_config = "proxy_read_timeout 300s;"
  }

  location {
    path           = "/static"
    forward_scheme = "https"
    forward_host   = "10.10.0.3"
    forward_port   = 443
  }
}
`, id, enabled, sslForced)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(false, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "domain_names.#", "3"),
					resource.TestCheckTypeSetElemAttr(name, "domain_names.*", fmt.Sprintf("zz_%d.example.com", id)),
					resource.TestCheckResourceAttr(name, "ssl_forced", "true"),
					resource.TestCheckResourceAttr(name, "hsts_enabled", "true"),
					resource.TestCheckResourceAttr(name, "trust_forwarded_proto", "true"),
					resource.TestCheckResourceAttr(name, "enabled", "false"),
					resource.TestCheckResourceAttr(name, "location.#", "2"),
					resource.TestCheckResourceAttr(name, "location.0.forward_path", "/v1"),
					resource.TestCheckResourceAttr(name, "location.0.advanced_config", "proxy_read_timeout 300s;"),
					resource.TestCheckResourceAttr(name, "location.1.forward_path", ""),
					checkProxyHostInNPM(t, name, func(h sdk.ProxyHost) error {
						if h.Enabled == nil || *h.Enabled {
							return fmt.Errorf("expected the host to be disabled in NPM")
						}
						if h.SSLForced == nil || !*h.SSLForced {
							return fmt.Errorf("expected ssl_forced to be set in NPM")
						}
						return nil
					}),
				),
			},
			// The same config again must not plan any changes.
			{
				Config: config(false, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Enable it and turn off SSL forcing in one update.
			{
				Config: config(true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttr(name, "ssl_forced", "false"),
					resource.TestCheckResourceAttr(name, "nginx_online", "true"),
					checkProxyHostInNPM(t, name, func(h sdk.ProxyHost) error {
						if h.Enabled == nil || !*h.Enabled {
							return fmt.Errorf("expected the host to be enabled in NPM")
						}
						return nil
					}),
				),
			},
			// Deleting the host outside Terraform plans to create it again.
			{
				Config: config(true, false),
				Check: deleteOutOfBand(t, name, func(c *sdk.Client, id int) error {
					return c.ProxyHosts.Delete(id)
				}),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config(true, false),
				Check:  resource.TestCheckResourceAttrSet(name, "id"),
			},
		},
	})
}

// NPM keeps a host whose config nginx rejects, and the provider reports it
// through nginx_online and nginx_err instead of failing.
func TestAccProxyHostResourceNginxOffline(t *testing.T) {

	var id = rand.IntN(1e4)
	const name = "nginxproxymanager_proxy_host.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host   = "10.10.0.1"
  forward_port   = 8080
  forward_scheme = "http"
  domain_names   = ["offline_%d.example.com"]

  location {
    path           = "/api"
    forward_scheme = "http"
    forward_host   = "does-not-resolve.invalid"
    forward_port   = 9000
  }
}
`, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "nginx_online", "false"),
					resource.TestMatchResourceAttr(name, "nginx_err", regexp.MustCompile("host not found in upstream")),
				),
			},
		},
	})
}

func checkProxyHostInNPM(t *testing.T, name string, check func(sdk.ProxyHost) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := resourceID(s, name)
		if err != nil {
			return err
		}
		host, err := newSDKClient(t).ProxyHosts.Get(id)
		if err != nil {
			return err
		}
		return check(host)
	}
}
