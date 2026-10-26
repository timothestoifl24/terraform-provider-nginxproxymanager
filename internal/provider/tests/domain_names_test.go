// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// NPM stores domain names sorted. As a list, any other order showed up as
// a change on every plan, and replaced Letsencrypt certificates.
func TestAccDomainNamesUnsorted(t *testing.T) {

	var id = rand.IntN(1e4)
	credentials := "dns_powerdns_api_url = http://pdns:8081\r\ndns_powerdns_api_key = npm"
	config := providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host   = "10.0.0.10"
  forward_port   = 8080
  forward_scheme = "http"
  domain_names   = ["zz_%[1]d.example.com", "aa_%[1]d.example.com"]
}

resource "nginxproxymanager_redirection_host" "test" {
  domain_names        = ["zz_redirect_%[1]d.example.com", "aa_redirect_%[1]d.example.com"]
  forward_domain_name = "target_%[1]d.example.com"
  forward_scheme      = "https"
  forward_http_code   = 301
}

resource "nginxproxymanager_dead_host" "test" {
  domain_names = ["zz_dead_%[1]d.example.com", "aa_dead_%[1]d.example.com"]
}

resource "nginxproxymanager_letsencrypt_certificate" "test" {
  domain_names             = ["www.example.com", "2.example.com"]
  dns_challenge            = true
  dns_provider             = "powerdns"
  dns_provider_credentials = base64decode("%[2]s")
  letsencrypt_email        = "test@test.com"
  propagation_seconds      = 5
}
`, id, base64.StdEncoding.EncodeToString([]byte(credentials)))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "domain_names.#", "2"),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_proxy_host.test", "domain_names.*", fmt.Sprintf("zz_%d.example.com", id)),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_letsencrypt_certificate.test", "domain_names.*", "www.example.com"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
