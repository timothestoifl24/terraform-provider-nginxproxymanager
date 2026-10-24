// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Redirection hosts used to drop ssl_forced on update and couldn't be
// disabled. Domains are given in an order NPM doesn't keep.
func TestAccRedirectionHostResourceRoundTrip(t *testing.T) {

	var id = rand.IntN(1e4)
	const name = "nginxproxymanager_redirection_host.test"
	certConfig := customCertificateConfig(fmt.Sprintf("cert_%d", id))

	config := func(enabled bool, sslForced bool) string {
		return providerConfig + certConfig + fmt.Sprintf(`
resource "nginxproxymanager_redirection_host" "test" {
  domain_names        = ["zz_%[1]d.example.com", "aa_%[1]d.example.com"]
  forward_domain_name = "target_%[1]d.example.com"
  forward_scheme      = "auto"
  forward_http_code   = 302
  preserve_path       = true
  certificate_id      = nginxproxymanager_custom_certificate.cert_%[1]d.id
  ssl_forced          = %[3]t
  enabled             = %[2]t
}
`, id, enabled, sslForced)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "enabled", "false"),
					resource.TestCheckResourceAttr(name, "ssl_forced", "false"),
				),
			},
			{
				Config: config(false, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: config(true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttr(name, "ssl_forced", "true"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config(true, true),
				Check: deleteOutOfBand(t, name, func(c *sdk.Client, id int) error {
					return c.RedirectionHosts.Delete(id)
				}),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Dead hosts used to never send ssl_forced, and couldn't be disabled.
func TestAccDeadHostResourceRoundTrip(t *testing.T) {

	var id = rand.IntN(1e4)
	const name = "nginxproxymanager_dead_host.test"
	certConfig := customCertificateConfig(fmt.Sprintf("cert_%d", id))

	config := func(enabled bool) string {
		return providerConfig + certConfig + fmt.Sprintf(`
resource "nginxproxymanager_dead_host" "test" {
  domain_names   = ["zz_%[1]d.example.com", "aa_%[1]d.example.com"]
  certificate_id = nginxproxymanager_custom_certificate.cert_%[1]d.id
  ssl_forced     = true
  hsts_enabled   = true
  enabled        = %[2]t
}
`, id, enabled)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "ssl_forced", "true"),
					resource.TestCheckResourceAttr(name, "hsts_enabled", "true"),
				),
			},
			{
				Config: config(false),
				Check:  resource.TestCheckResourceAttr(name, "enabled", "false"),
			},
			{
				Config: config(false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Streams had no enabled attribute.
func TestAccStreamResourceEnabled(t *testing.T) {

	port := 20000 + rand.IntN(1e4)
	const name = "nginxproxymanager_stream.test"

	config := func(enabled bool) string {
		return providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_stream" "test" {
  incoming_port   = %d
  forwarding_host = "10.10.0.1"
  forwarding_port = 5432
  enabled         = %t
}
`, port, enabled)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(false),
				Check:  resource.TestCheckResourceAttr(name, "enabled", "false"),
			},
			{
				Config: config(true),
				Check:  resource.TestCheckResourceAttr(name, "enabled", "true"),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config(true),
				Check: deleteOutOfBand(t, name, func(c *sdk.Client, id int) error {
					return c.Streams.Delete(id)
				}),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
