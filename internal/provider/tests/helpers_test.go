// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// newSDKClient returns a client for changing NPM behind Terraform's back.
func newSDKClient(t *testing.T) *sdk.Client {
	t.Helper()
	client, err := sdk.NewClientWithOptions(
		fmt.Sprintf("http://%s", providerHost),
		sdk.AuthStruct{Identity: providerUsername, Secret: providerPassword},
		sdk.ClientOptions{},
	)
	if err != nil {
		t.Fatalf("unable to create NPM client: %s", err)
	}
	return client
}

// resourceID returns the numeric NPM ID of a resource in state.
func resourceID(s *terraform.State, name string) (int, error) {
	rs, ok := s.RootModule().Resources[name]
	if !ok {
		return 0, fmt.Errorf("resource %s not found in state", name)
	}
	return strconv.Atoi(rs.Primary.ID)
}

// deleteOutOfBand deletes a resource directly in NPM, to check the provider
// notices and plans to create it again instead of failing.
func deleteOutOfBand(t *testing.T, name string, del func(*sdk.Client, int) error) func(*terraform.State) error {
	return func(s *terraform.State) error {
		id, err := resourceID(s, name)
		if err != nil {
			return err
		}
		return del(newSDKClient(t), id)
	}
}

// customCertificateConfig returns config for a self-signed custom
// certificate, for tests that need a certificate on a host.
func customCertificateConfig(name string) string {
	cert, key := sdk.NewCertWithKey()
	return fmt.Sprintf(`
resource "nginxproxymanager_custom_certificate" "%s" {
  nice_name       = "%s"
  certificate     = base64decode("%s")
  certificate_key = base64decode("%s")
}
`, name, name, base64.StdEncoding.EncodeToString([]byte(cert)), base64.StdEncoding.EncodeToString([]byte(key)))
}
