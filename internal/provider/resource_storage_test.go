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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	storageResourceName = "irmc-redfish_storage.sto"
)

func TestAccStorageResource_positive_complex(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageResourceConfig(creds, os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER"),
					"StopOnErrors", false, 33, 31, 32, 35, 240),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(storageResourceName, "storage_controller_serial_number",
						os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER")),
					resource.TestCheckResourceAttr(storageResourceName, "patrol_read_rate", "33"),
					resource.TestCheckResourceAttr(storageResourceName, "bios_continue_on_error", "StopOnErrors"),
					resource.TestCheckResourceAttr(storageResourceName, "bios_status", "false"),
					resource.TestCheckResourceAttr(storageResourceName, "bgi_rate", "31"),
					resource.TestCheckResourceAttr(storageResourceName, "mdc_rate", "32"),
					resource.TestCheckResourceAttr(storageResourceName, "rebuild_rate", "35"),
					resource.TestCheckResourceAttr(storageResourceName, "job_timeout", "240"),
				),
			},
			{
				Config: testAccStorageResourceConfig(creds, os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER"),
					"PauseOnErrors", true, 30, 32, 33, 34, 240),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(storageResourceName, "storage_controller_serial_number",
						os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER")),
					resource.TestCheckResourceAttr(storageResourceName, "patrol_read_rate", "30"),
					resource.TestCheckResourceAttr(storageResourceName, "bios_continue_on_error", "PauseOnErrors"),
					resource.TestCheckResourceAttr(storageResourceName, "bios_status", "true"),
					resource.TestCheckResourceAttr(storageResourceName, "bgi_rate", "32"),
					resource.TestCheckResourceAttr(storageResourceName, "mdc_rate", "33"),
					resource.TestCheckResourceAttr(storageResourceName, "rebuild_rate", "34"),
					resource.TestCheckResourceAttr(storageResourceName, "job_timeout", "240"),
				),
			},
		},
	})
}

func getStorageImportConfiguration(creds TestingServerCredentials, serial string) (string, error) {
	return fmt.Sprintf("{\"storage_controller_serial_number\":\"%s\", \"username\":\"%s\", \"password\":\"%s\", \"endpoint\":\"https://%s\", \"ssl_insecure\":true}",
		serial, creds.Username, creds.Password, creds.Endpoint), nil
}

func TestAccStorageResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:       `resource "irmc-redfish_storage" "sto" {}`,
				ResourceName: storageResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(d *terraform.State) (string, error) {
					return getStorageImportConfiguration(creds, os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER"))
				},
			},
		},
	})
}

func TestAccStorageResource_positive_simple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageResourceSimpleConfig(creds, os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER"), "PauseOnErrors", 31),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(storageResourceName, "storage_controller_serial_number", os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER")),
					resource.TestCheckResourceAttr(storageResourceName, "bios_continue_on_error", "PauseOnErrors"),
				),
			},
		},
	})
}

func TestAccStorageResource_negative_invalidServerSerial(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccStorageResourceSimpleConfig(creds, "qwerty", "PauseOnErrors", 30),
				ExpectError: regexp.MustCompile("Requested storage serial does not match to any installed controller serial."),
			},
		},
	})
}

func TestAccStorageResource_negative_tooShortTimeout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccStorageResourceConfig2(creds, os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER"), "Automatic", 30, 31, 32, 34, 0),
				ExpectError: regexp.MustCompile("Error while waiting for resource update."),
			},
		},
	})
}

func TestAccStorageResource_negative_emptyPayload(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccStorageResourceConfigEmptyPayload(creds, os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER")),
				ExpectError: regexp.MustCompile("Payload created out of defined plan will be empty."),
			},
		},
	})
}

func testAccStorageResourceSimpleConfig(testingInfo TestingServerCredentials, serial string, bios_continue_on_error string, bgi_rate int64) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_storage" "sto" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}

        storage_controller_serial_number = "%s"
		bios_continue_on_error = "%s"
		bgi_rate = %d
	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		serial,
		bios_continue_on_error,
		bgi_rate,
	)
}

func testAccStorageResourceConfig2(testingInfo TestingServerCredentials, serial string,
	patrol_read string, patrol_read_rate int64, bgi_rate, mdc_rate, rebuild_rate, job_timeout int64) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_storage" "sto" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}

        storage_controller_serial_number = "%s"
		bios_status = false
		patrol_read = "%s"
		patrol_read_rate = %d
		bgi_rate = %d
		mdc_rate = %d
		rebuild_rate = %d
        job_timeout = %d
	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		serial,
		patrol_read, patrol_read_rate, bgi_rate, mdc_rate, rebuild_rate, job_timeout,
	)
}

func testAccStorageResourceConfig(testingInfo TestingServerCredentials, serial string, bios_continue_on_error string, bios_status bool,
	patrol_read_rate int64, bgi_rate, mdc_rate, rebuild_rate, job_timeout int64) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_storage" "sto" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}

        storage_controller_serial_number = "%s"
        bios_continue_on_error = "%s"
        bios_status = %t
		patrol_read_rate = %d
		bgi_rate = %d
		mdc_rate = %d
		rebuild_rate = %d
        job_timeout = %d
	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		serial, bios_continue_on_error, bios_status,
		patrol_read_rate, bgi_rate, mdc_rate, rebuild_rate, job_timeout,
	)
}

func testAccStorageResourceConfigEmptyPayload(testingInfo TestingServerCredentials, serial string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_storage" "sto" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}

        storage_controller_serial_number = "%s"
	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		serial,
	)
}
