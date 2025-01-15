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
	storage_volume_resource_name = "irmc-redfish_storage_volume.volume"
)

// These tests are very hardware dependent (controller existence, id, disks etc.) so be.
func TestAccRedfishStorageVolume_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPrepareStorageVolume(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceStorageVolumeConfig_withCapacity(
					creds, os.Getenv("TF_TESTING_STORAGE_SERIAL_NUMBER"), "RAID0", 100000000, "my-name", 131072, "ReadAhead", "WriteThrough",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(storage_volume_resource_name, "name", "my-name"),
					resource.TestCheckResourceAttr(storage_volume_resource_name, "raid_type", "RAID0"),
					resource.TestCheckResourceAttr(storage_volume_resource_name, "read_mode.requested", "ReadAhead"),
					resource.TestCheckResourceAttr(storage_volume_resource_name, "write_mode.requested", "WriteThrough"),
				),
			},
		},
	})
}

func TestAccRedfishStorageVolume_InvalidStorageController(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPrepareStorageVolume(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceStorageVolumeConfig_withCapacity(
					creds, "qwerty", "RAID0", 100000000, "my-name", 131072, "ReadAhead", "WriteThrough",
				),
				ExpectError: regexp.MustCompile("Requested Storage resource has not been found on list"),
			},
		},
	})
}

func testAccRedfishResourceStorageVolumeConfig_withCapacity(testingInfo TestingServerCredentials,
	storage_controller_id string,
	raid_type string,
	capacity_bytes int64,
	name string,
	optimum_io_size_bytes int64,
	read_mode string,
	write_mode string,
) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_storage_volume" "volume" {
	  
		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        storage_controller_serial_number = "%s"
        raid_type = "%s"
        physical_drives = [ "[\"0\", \"3\"]" ]
        capacity_bytes = %d
        name = "%s"
        optimum_io_size_bytes = %d
        read_mode = {
            requested = "%s"
        }
        write_mode = {
            requested = "%s"
        }
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		storage_controller_id,
		raid_type,
		capacity_bytes,
		name,
		optimum_io_size_bytes,
		read_mode,
		write_mode,
	)
}
