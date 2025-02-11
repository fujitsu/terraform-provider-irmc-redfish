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

const (
	simpleUpdateResourceName = "irmc-redfish_simple_update.simple_update"
	TRANSFER_PROTOCOL        = "http"
	APPLY_TIME               = "Immediate"
)

func TestAccSimpleUpdateResource_correct(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSimpleUpdateResourceConfig(creds, TRANSFER_PROTOCOL, os.Getenv("TF_TESTING_SIMPLE_UPDATE_IMAGE_URL"), APPLY_TIME),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(simpleUpdateResourceName, "transfer_protocol", TRANSFER_PROTOCOL),
					resource.TestCheckResourceAttr(simpleUpdateResourceName, "update_image", os.Getenv("TF_TESTING_SIMPLE_UPDATE_IMAGE_URL")),
					resource.TestCheckResourceAttr(simpleUpdateResourceName, "operation_apply_time", APPLY_TIME),
				),
			},
		},
	})
}

func TestAccSimpleUpdateResource_invalidTransferProtocol(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSimpleUpdateResourceConfig(creds, "sftp", os.Getenv("TF_TESTING_SIMPLE_UPDATE_IMAGE_URL"), APPLY_TIME),
				ExpectError: regexp.MustCompile("Invalid Attribute Value Match"),
			},
		},
	})
}

func TestAccSimpleUpdateResource_missingImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSimpleUpdateResourceConfig(creds, TRANSFER_PROTOCOL, "", APPLY_TIME),
				ExpectError: regexp.MustCompile("Simple Update task did not complete successfully"),
			},
		},
	})
}

func TestAccSimpleUpdateResource_invalidApplyTime(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSimpleUpdateResourceConfig(creds, TRANSFER_PROTOCOL, os.Getenv("TF_TESTING_SIMPLE_UPDATE_IMAGE_URL"), "InvalidTime"),
				ExpectError: regexp.MustCompile("Invalid Attribute Value Match"),
			},
		},
	})
}

func testAccSimpleUpdateResourceConfig(testingInfo TestingServerCredentials, transferProtocol, updateImage, applyTime string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_simple_update" "simple_update" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}

		transfer_protocol  = "%s"
		update_image       = "%s"
		operation_apply_time = "%s"
	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		transferProtocol,
		updateImage,
		applyTime,
	)
}
