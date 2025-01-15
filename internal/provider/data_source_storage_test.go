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

func TestAccStorageDataSource_positive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageDataSourceConfig(creds, os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER")),
			},
		},
	})
}

func TestAccStorageDataSource_negative_invalidServerSerial(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccStorageDataSourceConfig(creds, "qwerty"),
				ExpectError: regexp.MustCompile("Could not obtain storage resource settings"),
			},
		},
	})
}

func testAccStorageDataSourceConfig(testingInfo TestingServerCredentials, serial string) string {
	return fmt.Sprintf(`
    data "irmc-redfish_storage" "sto" {
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
