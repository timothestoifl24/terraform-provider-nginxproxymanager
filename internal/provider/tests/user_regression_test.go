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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserResourceRoundTrip(t *testing.T) {

	var id = rand.IntN(1e4)
	const user = "nginxproxymanager_user.test"
	const permissions = "nginxproxymanager_user_permissions.test"

	config := fmt.Sprintf(`
resource "nginxproxymanager_user" "test" {
  email    = "regression_%[1]d@example.com"
  name     = "Regression %[1]d"
  nickname = "regression"
}

resource "nginxproxymanager_user_permissions" "test" {
  user_id = nginxproxymanager_user.test.id
  permissions {
    visibility   = "user"
    proxy_hosts  = "view"
    certificates = "hidden"
  }
}

resource "nginxproxymanager_user_password" "test" {
  user_id  = nginxproxymanager_user.test.id
  password = "regression-password"
}
`, id)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + regexp.MustCompile(`regression_`).ReplaceAllString(config, "Regression_"),
				ExpectError: regexp.MustCompile("must be lowercase"),
				PlanOnly:    true,
			},
			{
				Config: providerConfig + config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(permissions, "permissions.visibility", "user"),
					resource.TestCheckResourceAttr(permissions, "permissions.proxy_hosts", "view"),
				),
			},
			{
				ResourceName:      user,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:                         permissions,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "user_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[user].Primary.ID, nil
				},
			},
			// Deleting the user outside Terraform plans to create the user,
			// its permissions and its password again
			{
				Config: providerConfig + config,
				Check: deleteOutOfBand(t, user, func(c *sdk.Client, id int) error {
					return c.Users.Delete(id)
				}),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: providerConfig + config,
				Check:  resource.TestCheckResourceAttrSet(user, "id"),
			},
		},
	})
}
