// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomCertificateResource(t *testing.T) {

	id := rand.IntN(1e4)
	name := fmt.Sprintf("test_%d", id)
	cert, key := sdk.NewCertWithKey()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
	resource "nginxproxymanager_custom_certificate" "test" {
	  nice_name       = "%s"
	  certificate     = base64decode("%s")
	  certificate_key = base64decode("%s")
	}
	`, name, base64.StdEncoding.EncodeToString([]byte(cert)), base64.StdEncoding.EncodeToString([]byte(key))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_custom_certificate.test", "id"),
					resource.TestCheckResourceAttr("nginxproxymanager_custom_certificate.test", "nice_name", name),
					resource.TestCheckResourceAttr("nginxproxymanager_custom_certificate.test", "certificate", cert),
					resource.TestCheckResourceAttr("nginxproxymanager_custom_certificate.test", "certificate_key", key),
				),
			},
			{
				ResourceName:      "nginxproxymanager_custom_certificate.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Deleting the custom certificate outside Terraform, e.g in the UI, makes
			// Terraform plan to create it again instead of failing
			{
				Config: providerConfig + fmt.Sprintf(`
	resource "nginxproxymanager_custom_certificate" "test" {
	  nice_name       = "%s"
	  certificate     = base64decode("%s")
	  certificate_key = base64decode("%s")
	}
	`, name, base64.StdEncoding.EncodeToString([]byte(cert)), base64.StdEncoding.EncodeToString([]byte(key))),
				Check:              deleteInNPM("nginxproxymanager_custom_certificate.test", func(c *sdk.Client, id int) error { return c.Certificates.Delete(id) }),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
