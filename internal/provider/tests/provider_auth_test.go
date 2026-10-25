// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The provider can authenticate with only a token, e.g for a user with
// two-factor authentication.
func TestAccProviderTokenAuth(t *testing.T) {

	client := newSDKClient(t)
	if err := client.GetToken(); err != nil {
		t.Fatalf("unable to get a token: %s", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// The last step's config is used to destroy, so the failing one is first
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "nginxproxymanager" {
  host = "%s"
}

data "nginxproxymanager_version" "current" {}
`, providerHost),
				ExpectError: regexp.MustCompile("Missing Nginx Proxy Manager credentials"),
			},
			{
				Config: fmt.Sprintf(`
provider "nginxproxymanager" {
  host  = "%s"
  token = "%s"
}

data "nginxproxymanager_version" "current" {}
`, providerHost, client.Token),
				Check: resource.TestCheckResourceAttr("data.nginxproxymanager_version.current", "major", "2"),
			},
		},
	})
}
