// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDefaultSiteResource(t *testing.T) {

	const name = "nginxproxymanager_default_site.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "nginxproxymanager_default_site" "test" {
  page = "redirect"
}
`,
				ExpectError: regexp.MustCompile("redirect must be set"),
			},
			{
				Config: providerConfig + `
resource "nginxproxymanager_default_site" "test" {
  page     = "redirect"
  redirect = "https://example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "default-site"),
					resource.TestCheckResourceAttr(name, "page", "redirect"),
					resource.TestCheckResourceAttr(name, "redirect", "https://example.com"),
				),
			},
			{
				Config: providerConfig + `
resource "nginxproxymanager_default_site" "test" {
  page = "html"
  html = "<h1>Nothing here</h1>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "page", "html"),
					resource.TestCheckNoResourceAttr(name, "redirect"),
				),
			},
			{
				Config: providerConfig + `
resource "nginxproxymanager_default_site" "test" {
  page = "html"
  html = "<h1>Nothing here</h1>"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateId:     "default-site",
				ImportStateVerify: true,
			},
		},
		// Destroying it resets NPM to the congratulations page
		CheckDestroy: func(_ *terraform.State) error {
			setting, err := newSDKClient(t).Settings.Get("default-site")
			if err != nil {
				return err
			}
			if *setting.Value != "congratulations" {
				return fmt.Errorf("expected the default site to be reset, got %s", *setting.Value)
			}
			return nil
		},
	})
}

func TestAccDNSProvidersAndVersionDataSources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "nginxproxymanager_dns_providers" "all" {}
data "nginxproxymanager_version" "current" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("data.nginxproxymanager_dns_providers.all", "providers.*", map[string]string{
						"id": "hetzner",
					}),
					resource.TestMatchResourceAttr("data.nginxproxymanager_version.current", "version", regexp.MustCompile(`^2\.\d+\.\d+$`)),
					resource.TestCheckResourceAttr("data.nginxproxymanager_version.current", "major", "2"),
				),
			},
		},
	})
}
