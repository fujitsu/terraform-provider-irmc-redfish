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
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stmcginnis/gofish"
)

const (
	resource_irmc_host_power = "irmc-redfish_power.pwr"
	sleepDuration            = 5 * time.Minute
)

func TestAccRedfishIrmcPower(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			clientConfig := gofish.ClientConfig{
				Endpoint:  "https://" + creds.Endpoint,
				Username:  creds.Username,
				Password:  creds.Password,
				BasicAuth: true,
				Insecure:  true,
			}
			api, err := gofish.Connect(clientConfig)
			if err != nil {
				t.Fatalf("Failed to connect to %s: %s", clientConfig.Endpoint, err.Error())
			}
			defer api.Logout()

			isPoweredOn, err := isPoweredOn(api.Service)
			if err != nil {
				t.Fatalf("Failed to check power state: %s", err.Error())
			}

			if isPoweredOn {
				if err = changePowerState(api.Service, false, 120); err != nil {
					t.Fatalf("Failed to change power state within given timeout: %s", err.Error())
				}
			}
		},
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourcePowerConfig(creds, "On"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_irmc_host_power, "power_state", "On"),
				),
			},
			{
				PreConfig: func() {
					time.Sleep(sleepDuration)
				},
				Config: testAccRedfishResourcePowerConfig(creds, "GracefulShutdown"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_irmc_host_power, "power_state", "Off"),
				),
			},
			{
				Config: testAccRedfishResourcePowerConfig(creds, "ForceOn"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_irmc_host_power, "power_state", "On"),
				),
			},
			{
				Config: testAccRedfishResourcePowerConfig(creds, "ForceOff"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_irmc_host_power, "power_state", "Off"),
				),
			},
			{
				Config: testAccRedfishResourcePowerConfig(creds, "On"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_irmc_host_power, "power_state", "On"),
				),
			},
			{
				PreConfig: func() {
					time.Sleep(sleepDuration)
				},
				Config: testAccRedfishResourcePowerConfig(creds, "ForceRestart"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_irmc_host_power, "power_state", "On"),
				),
			},
			{
				PreConfig: func() {
					time.Sleep(sleepDuration)
				},
				Config: testAccRedfishResourcePowerConfig(creds, "PowerCycle"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_irmc_host_power, "power_state", "On"),
				),
			},
			{
				// This test case might lead to problems when booted host OS does not have
				// configured behavior for power button (e.g.: in Linux environment)
				Config: testAccRedfishResourcePowerConfig(creds, "PushPowerButton"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_irmc_host_power, "power_state", "Off"),
				),
			},
		},
	})
}

func testAccRedfishResourcePowerConfig(testingInfo TestingServerCredentials,
	HostPowerAction string,
) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_power" "pwr" {
	  
		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}
		  host_power_action = "%s"
		  max_wait_time = 120
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		HostPowerAction,
	)
}
