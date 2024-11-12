/*
Copyright (c) 2024 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const bo_name = "irmc-redfish_boot_order.bo"

func TestAccRedfishBootOrder_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { testChangePowerHostState(creds, true) },
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(bo_name, "id", "/redfish/v1/Systems/0/Bios/Settings"),
					resource.TestCheckResourceAttr(bo_name, "system_reset_type", "ForceRestart"),
				),
			},
			{
				PreConfig: func() { testChangePowerHostState(creds, false) },
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST_OPPOSITE"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(bo_name, "id", "/redfish/v1/Systems/0/Bios/Settings"),
					resource.TestCheckResourceAttr(bo_name, "system_reset_type", "ForceRestart"),
				),
			},
		},
	})
}

func TestAccRedfishBootOrder_negative_wrongBootOrder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST_TOO_SHORT"),
				),
				ExpectError: regexp.MustCompile("Planned boot order has different length than currently configured boot order"),
			},
			{
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST_DUPLICATED"),
				),
				ExpectError: regexp.MustCompile("Planned boot order does not contain all available boot options"),
			},
			{
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST_WRONG_BOOT_ENTRY"),
				),
				ExpectError: regexp.MustCompile("Planned changes for boot order did not pass validation"),
			},
		},
	})
}

func testAccRedfishResourceBootOrderConfig(testingInfo TestingServerCredentials,
	boot_order string,
) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_boot_order" "bo" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        boot_order = %s
        system_reset_type = "ForceRestart"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		boot_order,
	)
}
