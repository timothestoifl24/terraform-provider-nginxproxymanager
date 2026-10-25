// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Importing a certificate and applying the config it came from must not
// issue a new certificate, even though NPM doesn't return the DNS provider
// credentials. Domains are given in an order NPM doesn't keep.
func TestAccLetsencryptCertificateResourceImportNoReplace(t *testing.T) {

	const name = "nginxproxymanager_letsencrypt_certificate.test"
	const imported = "nginxproxymanager_letsencrypt_certificate.imported"
	credentials := "dns_powerdns_api_url = http://pdns:8081\r\ndns_powerdns_api_key = npm"
	certificate := func(name string) string {
		return fmt.Sprintf(`
resource "nginxproxymanager_letsencrypt_certificate" "%s" {
  domain_names             = ["www.example.com", "2.example.com"]
  dns_challenge            = true
  dns_provider             = "powerdns"
  dns_provider_credentials = base64decode("%s")
  propagation_seconds      = 5
}
`, name, base64.StdEncoding.EncodeToString([]byte(credentials)))
	}
	config := providerConfig + certificate("test")
	// The same certificate imported a second time, as it would be by
	// someone starting to manage an existing NPM with Terraform
	importConfig := config + certificate("imported")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(name, "domain_names.#", "2"),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config:             importConfig,
				ResourceName:       imported,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[name].Primary.ID, nil
				},
			},
			// The imported certificate has no credentials in state, which
			// must be an update, not a replacement.
			{
				Config: importConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(imported, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: importConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// The DNS provider is checked against the providers NPM reports, which used
// to be a fixed list that rejected e.g hetzner and ovh.
func TestAccLetsencryptCertificateResourceDNSProviders(t *testing.T) {

	config := func(provider string) string {
		return providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_letsencrypt_certificate" "test" {
  domain_names             = ["dns-provider-check.example.com"]
  dns_challenge            = true
  dns_provider             = "%s"
  dns_provider_credentials = "dns_example_token = abc"
}
`, provider)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             config("hetzner"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:      config("not-a-dns-provider"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`doesn't support the DNS provider "not-a-dns-provider"`),
			},
		},
	})
}

// A custom certificate without a nice_name used to fail with an inconsistent
// result, as NPM returns one anyway.
func TestAccCustomCertificateResourceNoNiceName(t *testing.T) {

	const name = "nginxproxymanager_custom_certificate.test"
	config := providerConfig + customCertificateConfig("test")
	// Remove the nice_name from the shared config
	config = regexp.MustCompile(`(?m)^\s*nice_name.*$`).ReplaceAllString(config, "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(name, "nice_name", ""),
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
