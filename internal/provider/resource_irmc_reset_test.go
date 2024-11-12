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
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stmcginnis/gofish"
)

// Test to create irmc reset resource with invalid id.
func TestAccRedfishIRMCReset_Invalid_ResetType_Negative(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRedfishResourceIRMCResetConfig(creds, "iRMCs"),
				ExpectError: regexp.MustCompile("Invalid IRMC ID provided"),
			},
		},
	})
}

// Test to perform irmc reset when host on

func TestAccRedfishIRMCReset_HostOn(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					err := testpreIrmcHostPowerOn(creds, true)
					if err != nil {
						t.Fatalf("Error during pre-configuration: %s", err)
					}

				},
				Config: testAccRedfishResourceIRMCResetConfig(creds, "iRMC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("irmc-redfish_irmc_reset.irmc_rst", "id", "iRMC"),
				),
			},
		},
	})
}

// Test to perform irmc reset when host off.
func TestAccRedfishIRMCReset_HostOff(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					err := testpreIrmcHostPowerOn(creds, false)
					if err != nil {
						t.Fatalf("Error during pre-configuration: %s", err)
					}
				},
				Config: testAccRedfishResourceIRMCResetConfig(creds, "iRMC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("irmc-redfish_irmc_reset.irmc_rst", "id", "iRMC"),
				),
			},
		},
	})
}

func testAccRedfishResourceIRMCResetConfig(testingInfo TestingServerCredentials,
	id string,
) string {
	return fmt.Sprintf(`
		
	resource "irmc-redfish_irmc_reset" "irmc_rst" {
		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}
		id = "%s"
	}
		`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		id,
	)
}

func testpreIrmcHostPowerOn(creds TestingServerCredentials, hoston bool) error {
	clientConfig := gofish.ClientConfig{
		Endpoint:  "https://" + creds.Endpoint,
		Username:  creds.Username,
		Password:  creds.Password,
		BasicAuth: true,
		Insecure:  true,
	}

	api, err := gofish.Connect(clientConfig)
	if err != nil {
		return err
	}
	defer api.Logout()
	isPoweredOn, err := isPoweredOn(api.Service)
	if err != nil {
		return err
	}
	if hoston && !isPoweredOn {
		err = changePowerState(api.Service, true, 300)
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Minute)
	} else if !hoston && isPoweredOn {
		err = changePowerState(api.Service, false, 300)
		if err != nil {
			return err
		}
		time.Sleep(45 * time.Second)
	}

	return nil
}
