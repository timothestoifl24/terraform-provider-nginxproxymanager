// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Proxy hosts with location blocks failed to load, as the location model
// had a domain_names field the schema doesn't have.
func TestAccProxyHostResourceLocations(t *testing.T) {

	var id = rand.IntN(1e4)
	config := providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host   = "10.0.0.10"
  forward_port   = 8080
  forward_scheme = "http"
  domain_names   = ["locations_%d.example.com"]

  location {
    path           = "/api"
    forward_scheme = "http"
    forward_host   = "10.0.0.11"
    forward_port   = 9000
  }

  location {
    path           = "/static"
    forward_scheme = "https"
    forward_host   = "10.0.0.12"
    forward_port   = 443
  }
}

data "nginxproxymanager_proxy_host" "test" {
  id = nginxproxymanager_proxy_host.test.id
}
`, id)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.#", "2"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.0.path", "/api"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.1.forward_port", "443"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "location.#", "2"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "location.0.forward_host", "10.0.0.11"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      "nginxproxymanager_proxy_host.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
