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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAccessListResourceRoundTrip(t *testing.T) {

	var id = rand.IntN(1e4)
	const name = "nginxproxymanager_access_list.test"
	const imported = "nginxproxymanager_access_list.imported"

	accessList := func(resourceName, listName string) string {
		return fmt.Sprintf(`
resource "nginxproxymanager_access_list" "%s" {
  name        = "%s"
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
`, resourceName, listName)
	}
	config := func(listName string) string {
		return providerConfig + accessList("test", listName)
	}
	listName := fmt.Sprintf("test_%d", id)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(listName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "item.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(name, "item.*", map[string]string{"username": "bob"}),
					resource.TestCheckResourceAttr(name, "client.#", "1"),
				),
			},
			{
				Config: config(listName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// Renaming the list must keep its users and settings
			{
				Config: config(listName + "_renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", listName+"_renamed"),
					resource.TestCheckResourceAttr(name, "satisfy_any", "true"),
					checkAccessListInNPM(t, name, func(a sdk.AccessList) error {
						if a.Items == nil || len(*a.Items) != 2 {
							return fmt.Errorf("expected 2 users in NPM after renaming")
						}
						return nil
					}),
				),
			},
			// A user added outside Terraform shows up as a change
			{
				Config: config(listName + "_renamed"),
				Check: func(s *terraform.State) error {
					id, err := resourceID(s, name)
					if err != nil {
						return err
					}
					client := newSDKClient(t)
					current, err := client.AccessLists.Get(id)
					if err != nil {
						return err
					}
					items := []sdk.AccessListItem{}
					for _, item := range *current.Items {
						// An empty password keeps the existing one
						items = append(items, sdk.AccessListItem{Username: item.Username, Password: new(string)})
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
				Config: config(listName + "_renamed"),
				Check:  resource.TestCheckResourceAttr(name, "item.#", "2"),
			},
			// Importing it gives an in-place update to set the passwords,
			// then no further changes
			{
				Config:             config(listName+"_renamed") + accessList("imported", listName+"_renamed"),
				ResourceName:       imported,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[name].Primary.ID, nil
				},
			},
			{
				Config: config(listName+"_renamed") + accessList("imported", listName+"_renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(imported, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: config(listName+"_renamed") + accessList("imported", listName+"_renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// An access list with only client rules and no users.
func TestAccAccessListResourceClientsOnly(t *testing.T) {

	var id = rand.IntN(1e4)
	const name = "nginxproxymanager_access_list.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
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
`, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "item.#", "0"),
					resource.TestCheckResourceAttr(name, "client.#", "2"),
					resource.TestCheckResourceAttr(name, "client.1.address", "all"),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
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
`, id),
				Check: deleteOutOfBand(t, name, func(c *sdk.Client, id int) error {
					return c.AccessLists.Delete(id)
				}),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func checkAccessListInNPM(t *testing.T, name string, check func(sdk.AccessList) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := resourceID(s, name)
		if err != nil {
			return err
		}
		accessList, err := newSDKClient(t).AccessLists.Get(id)
		if err != nil {
			return err
		}
		return check(accessList)
	}
}
