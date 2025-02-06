/*
Copyright (c) 2025 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

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

func TestAccFirmwareUpdateResource_correct_MemoryCard_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirmwareUpdateResourceConfig(creds, "MemoryCard", "", "", ""),
			},
		},
	})
}

func TestAccFirmwareUpdateResource_correct_TFTP_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirmwareUpdateResourceConfig(creds, "TFTP", "", "10.172.181.125", "irmc/RX2530M7/RX2530M7_02.58e_sdr03.83.bin"),
			},
		},
	})
}

func TestAccFirmwareUpdateResource_correct_File_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirmwareUpdateResourceConfig(creds, "File", "/home/polecp/terraform/RX2530M7_02.58c_sdr03.83.bin", "", ""),
			},
		},
	})
}

func TestAccFirmwareUpdateResource_missingAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccFirmwareUpdateResourceConfig(creds, "File", "", "", ""),
				ExpectError: regexp.MustCompile("Field 'irmc_file_name' is required when 'update_type' equals 'File'."),
			},
		},
	})
}

func TestAccFirmwareUpdateResource_wrongUpdateFile_notBINfile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccFirmwareUpdateResourceConfig(creds, "File", "test.pem", "", ""),
				ExpectError: regexp.MustCompile("File firmware update failed."),
			},
		},
	})
}

func TestAccFirmwareUpdateResource_wrongUpdateFile_wrongPlatform(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccFirmwareUpdateResourceConfig(creds, "File", "/home/polecp/terraform/D3931_01.29S_sdr03.50.bin", "", ""),
				ExpectError: regexp.MustCompile("File Firmware Update task did not complete successfully"),
			},
		},
	})
}

func TestAccFirmwareUpdateResource_wrongUpdateTFTP_wrongtftpUpdateFile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccFirmwareUpdateResourceConfig(creds, "TFTP", "", "10.172.181.125", "irmc/RX2530M7/RX2530M9_02.58f_sdr03.83.bin"),
				ExpectError: regexp.MustCompile("TFTP Firmware Update task did not complete successfully"),
			},
		},
	})
}

func testAccFirmwareUpdateResourceConfig(testingInfo TestingServerCredentials, updateType, irmcPathToBinary, tftpServerAddrr, tftpUpdateFile string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_irmc_firmware_update" "irmcfu" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}

		update_type  = "%s"
		irmc_path_to_binary       = "%s"
		tftp_server_addr = "%s"
        tftp_update_file = "%s"


	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		updateType,
		irmcPathToBinary,
		tftpServerAddrr,
		tftpUpdateFile,
	)
}
