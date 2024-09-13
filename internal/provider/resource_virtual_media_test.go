package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	working_cdimage_path = "http://10.172.181.125:8006/gauge/vmedia/Cd!123.iso"
	working_hdimage_path = "http://10.172.181.125:8006/gauge/vmedia/Hd!123.img"
	resource_name        = "irmc-redfish_virtual_media.vm"
)

func getVMediaImportConfiguration(d *terraform.State, creds TestingServerCredentials) (string, error) {
	id := "/redfish/v1/Managers/iRMC/VirtualMedia/0"
	return fmt.Sprintf("{\"id\":\"%s\", \"username\":\"%s\", \"password\":\"%s\", \"endpoint\":\"https://%s\", \"ssl_insecure\":true}",
		id, creds.Username, creds.Password, creds.Endpoint), nil
}

func getVMediaImportHdConfiguration(d *terraform.State, creds TestingServerCredentials) (string, error) {
	id := "/redfish/v1/Managers/iRMC/VirtualMedia/1"
	return fmt.Sprintf("{\"id\":\"%s\", \"username\":\"%s\", \"password\":\"%s\", \"endpoint\":\"https://%s\", \"ssl_insecure\":true}",
		id, creds.Username, creds.Password, creds.Endpoint), nil
}

func getVMediaImportConfigurationInvalidId(d *terraform.State, creds TestingServerCredentials) (string, error) {
	return fmt.Sprintf("{\"id\":\"unknown\", \"username\":\"%s\", \"password\":\"%s\", \"endpoint\":\"https://%s\", \"ssl_insecure\":true}",
		creds.Username, creds.Password, creds.Endpoint), nil
}

func TestAccRedfishVirtualMedia_basic_cd(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPrepareVMediaSlots(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, "10.172.181.125/gauge/vmedia/Cd!123.iso", "NFS",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_name, "image", "10.172.181.125/gauge/vmedia/Cd!123.iso"),
					resource.TestCheckResourceAttr(resource_name, "inserted", "true"),
				),
			},
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, working_cdimage_path, "HTTP",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_name, "image", working_cdimage_path),
					resource.TestCheckResourceAttr(resource_name, "inserted", "true"),
				),
			},
			{
				ResourceName:      resource_name,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(d *terraform.State) (string, error) {
					return getVMediaImportConfiguration(d, creds)
				},
			},
			{
				ResourceName: resource_name,
				ImportState:  true,
				//                ImportStateVerify: true,
				ImportStateIdFunc: func(d *terraform.State) (string, error) {
					return getVMediaImportConfigurationInvalidId(d, creds)
				},
				ExpectError: regexp.MustCompile("Virtual media with ID unknown does not exist."),
			},
		},
	})
}

func TestAccRedfishVirtualMedia_basic_hd(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPrepareVMediaSlots(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, working_hdimage_path, "HTTP",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_name, "image", working_hdimage_path),
					resource.TestCheckResourceAttr(resource_name, "inserted", "true"),
				),
			},
			{
				ResourceName:      resource_name,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(d *terraform.State) (string, error) {
					return getVMediaImportHdConfiguration(d, creds)
				},
			},
		},
	})
}

func TestAccRedfishVirtualMedia_NotAllowedExtension(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPrepareVMediaSlots(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, "http://10.172.181.125:8006/gauge/vmedia/Cd!123.iso2", "HTTP",
				),
				ExpectError: regexp.MustCompile("Image type format is not supported"),
			},
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, "http://10.172.181.125:8006/gauge/vmedia/Hd!123.ima", "HTTP",
				),
				ExpectError: regexp.MustCompile("Image type format is not supported"),
			},
		},
	})
}

func testAccRedfishResourceVirtualMediaConfig(testingInfo TestingServerCredentials,
	image string,
	transfer_protocol_type string,
) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_virtual_media" "vm" {
	  
		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        image = "%s"
        transfer_protocol_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		image,
		transfer_protocol_type,
	)
}
