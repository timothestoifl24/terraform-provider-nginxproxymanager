// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAccessListResource(t *testing.T) {

	id := rand.IntN(1e4)
	name := fmt.Sprintf("test_%d", id)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_access_list" "test" {
  name = "%s"
  satisfy_any = true
  pass_auth = false

  item {
    username = "username1"
	password = "password1"
  }

  item {
    username = "username2"
	password = "password2"
  }

  client {
    address = "1.2.3.4"
	directive = "allow" 
  }

  client {
    address = "2.3.4.5"
	directive = "deny" 
  }

  client {
    address = "3.4.5.6"
	directive = "allow" 
  }
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_access_list.test", "id"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "name", name),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "item.#", "2"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "client.#", "3"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "satisfy_any", "true"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "pass_auth", "false"),
				),
			},
			{
				ResourceName:      "nginxproxymanager_access_list.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"item",
				},
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_access_list" "test" {
  name = "%s"

  item {
    username = "username1"
	password = "password1"
  }

  item {
    username = "username2"
	password = "password2"
  }

  item {
    username = "username3"
	password = "password3"
  }

  client {
    address = "1.2.3.4"
	directive = "allow" 
  }

  client {
    address = "3.4.5.6"
	directive = "allow" 
  }
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "name", name),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "item.#", "3"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "client.#", "2"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "satisfy_any", "false"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "pass_auth", "true"),
				),
			},
			// Deleting the access list outside Terraform, e.g in the UI, makes
			// Terraform plan to create it again instead of failing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_access_list" "test" {
  name = "%s"

  item {
    username = "username1"
	password = "password1"
  }

  item {
    username = "username2"
	password = "password2"
  }

  item {
    username = "username3"
	password = "password3"
  }

  client {
    address = "1.2.3.4"
	directive = "allow" 
  }

  client {
    address = "3.4.5.6"
	directive = "allow" 
  }
}`, name),
				Check:              deleteInNPM("nginxproxymanager_access_list.test", func(c *sdk.Client, id int) error { return c.AccessLists.Delete(id) }),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
