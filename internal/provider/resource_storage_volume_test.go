package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	// "github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
    storage_volume_resource_name = "irmc-redfish_storage_volume.volume"
)
                    
func TestAccRedfishStorageVolume_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPrepareStorageVolume(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceStorageVolumeConfig_withCapacity(
                    creds, "0", "RAID0", 100000000, "my-name", 131072, "ReadAhead", "WriteThrough",
                ),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr(storage_volume_resource_name, "name", "my-name"),
                    resource.TestCheckResourceAttr(storage_volume_resource_name, "raid_type", "RAID0"),
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
                    creds, "99", "RAID0", 100000000, "my-name", 131072, "ReadAhead", "WriteThrough",
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

        storage_controller_id = "%s"
        raid_type = "%s"
        physical_drives = [ "[\"0\", \"3\"]" ]
        capacity_bytes = %d
        name = "%s"
        optimum_io_size_bytes = %d
        read_mode = "%s"
        write_mode = "%s"
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
