// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The users of an access list weren't read back from NPM, so an imported
// list had no users and its generated config couldn't be applied, and users
// changed outside Terraform weren't noticed.
func TestAccAccessListResourceImport(t *testing.T) {

	var id = rand.IntN(1e4)
	const name = "nginxproxymanager_access_list.test"
	const imported = "nginxproxymanager_access_list.imported"

	accessList := func(resourceName string) string {
		return fmt.Sprintf(`
resource "nginxproxymanager_access_list" "%s" {
  name        = "import_%d"
  satisfy_any = true
  pass_auth   = false

  item {
    username = "alice"
    password = "alice-password"
  }

  item {
    username = "bob"
    password = "bob-password"
  }

  client {
    address   = "10.0.0.0/8"
    directive = "allow"
  }
}
`, resourceName, id)
	}
	config := providerConfig + accessList("test")
	importConfig := config + accessList("imported")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "item.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(name, "item.*", map[string]string{"username": "bob"}),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// A user added outside Terraform shows up as a change
			{
				Config: config,
				Check: func(s *terraform.State) error {
					id, err := strconv.Atoi(s.RootModule().Resources[name].Primary.ID)
					if err != nil {
						return err
					}
					client := sdk.NewClient(fmt.Sprintf("http://%s", providerHost), sdk.AuthStruct{Identity: providerUsername, Secret: providerPassword})
					if err := client.GetToken(); err != nil {
						return err
					}
					current, err := client.AccessLists.Get(id)
					if err != nil {
						return err
					}
					// Usernames without a password keep their current one
					items := []sdk.AccessListItem{}
					for _, item := range *current.Items {
						items = append(items, sdk.AccessListItem{Username: item.Username})
					}
					username, password := "mallory", "mallory-password"
					items = append(items, sdk.AccessListItem{Username: &username, Password: &password})
					update := current.AccessListBase
					update.Items = &items
					_, err = client.AccessLists.Update(id, update)
					return err
				},
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(name, "item.#", "2"),
			},
			// Importing it gives an in-place update to set the passwords,
			// then no further changes
			{
				Config:             importConfig,
				ResourceName:       imported,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[name].Primary.ID, nil
				},
			},
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

// An access list with only client rules and no users couldn't be created.
func TestAccAccessListResourceClientsOnly(t *testing.T) {

	var id = rand.IntN(1e4)
	config := providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_access_list" "test" {
  name = "clients_only_%d"

  client {
    address   = "192.168.0.0/16"
    directive = "allow"
  }

  client {
    address   = "all"
    directive = "deny"
  }
}
`, id)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "item.#", "0"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "client.1.address", "all"),
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
