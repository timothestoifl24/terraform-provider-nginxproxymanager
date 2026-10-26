// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// A custom certificate without a nice_name failed with an inconsistent
// result, as NPM returns an empty one.
func TestAccCustomCertificateResourceNoNiceName(t *testing.T) {

	cert, key := sdk.NewCertWithKey()
	config := providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_custom_certificate" "test" {
  certificate     = base64decode("%s")
  certificate_key = base64decode("%s")
}
`, base64.StdEncoding.EncodeToString([]byte(cert)), base64.StdEncoding.EncodeToString([]byte(key)))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr("nginxproxymanager_custom_certificate.test", "nice_name", ""),
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
