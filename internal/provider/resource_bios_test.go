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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const bios_name = "irmc-redfish_bios.bios"

func TestAccRedfishBios(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { testChangePowerHostState(creds, true) },
				Config: testAccRedfishResourceBiosConfig_correctAttributes(
					creds, "ForceRestart",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(bios_name, "id", "/redfish/v1/Systems/0/Bios/Settings"),
					resource.TestCheckResourceAttr(bios_name, "system_reset_type", "ForceRestart"),
				),
			},
		},
	})
}

func TestAccRedfishBios_negative(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRedfishResourceBiosConfig_wrongAttribute(creds, "ForceRestart"),
				ExpectError: regexp.MustCompile("Attribute 'XXX' is not supported by the system"),
			},
			{
				Config:      testAccRedfishResourceBiosConfig_notSupportedBootSources(creds, "ForceRestart"),
				ExpectError: regexp.MustCompile("Attribute 'BootSources' is not supported by the resource"),
			},
		},
	})
}

func testAccRedfishResourceBiosConfig_correctAttributes(testingInfo TestingServerCredentials, reset_type string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_bios" "bios" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "AssetTag": "TestAssetTag"
        }
        system_reset_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		reset_type,
	)
}

func testAccRedfishResourceBiosConfig_wrongAttribute(testingInfo TestingServerCredentials, reset_type string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_bios" "bios" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "XXX": "YYY"
        }
        system_reset_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		reset_type,
	)
}

func testAccRedfishResourceBiosConfig_notSupportedBootSources(testingInfo TestingServerCredentials, reset_type string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_bios" "bios" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "BootSources": "YYY"
        }
        system_reset_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		reset_type,
	)
}
