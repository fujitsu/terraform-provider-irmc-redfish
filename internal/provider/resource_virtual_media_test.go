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
	resource_name = "irmc-redfish_virtual_media.vm"
)

func getVMediaImportConfiguration(creds TestingServerCredentials) (string, error) {
	id := "/redfish/v1/Managers/iRMC/VirtualMedia/0"
	return fmt.Sprintf("{\"id\":\"%s\", \"username\":\"%s\", \"password\":\"%s\", \"endpoint\":\"https://%s\", \"ssl_insecure\":true}",
		id, creds.Username, creds.Password, creds.Endpoint), nil
}

func getVMediaImportHdConfiguration(creds TestingServerCredentials) (string, error) {
	id := "/redfish/v1/Managers/iRMC/VirtualMedia/1"
	return fmt.Sprintf("{\"id\":\"%s\", \"username\":\"%s\", \"password\":\"%s\", \"endpoint\":\"https://%s\", \"ssl_insecure\":true}",
		id, creds.Username, creds.Password, creds.Endpoint), nil
}

func getVMediaImportConfigurationInvalidId(creds TestingServerCredentials) (string, error) {
	return fmt.Sprintf("{\"id\":\"unknown\", \"username\":\"%s\", \"password\":\"%s\", \"endpoint\":\"https://%s\", \"ssl_insecure\":true}",
		creds.Username, creds.Password, creds.Endpoint), nil
}

func TestAccRedfishVirtualMedia_basic_cd_nfs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPrepareVMediaSlots(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, os.Getenv("TF_TESTING_VMEDIA_CD_PATH_NFS"), "NFS",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_name, "image", os.Getenv("TF_TESTING_VMEDIA_CD_PATH_NFS")),
					resource.TestCheckResourceAttr(resource_name, "inserted", "true"),
				),
			},
			{
				ResourceName:      resource_name,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(d *terraform.State) (string, error) {
					return getVMediaImportConfiguration(creds)
				},
			},
			{
				ResourceName: resource_name,
				ImportState:  true,
				//                ImportStateVerify: true,
				ImportStateIdFunc: func(d *terraform.State) (string, error) {
					return getVMediaImportConfigurationInvalidId(creds)
				},
				ExpectError: regexp.MustCompile("Virtual media with ID unknown does not exist."),
			},
		},
	})
}

func TestAccRedfishVirtualMedia_basic_cd_cifs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPrepareVMediaSlots(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, os.Getenv("TF_TESTING_VMEDIA_CD_PATH_CIFS"), "CIFS",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_name, "image", os.Getenv("TF_TESTING_VMEDIA_CD_PATH_CIFS")),
					resource.TestCheckResourceAttr(resource_name, "inserted", "true"),
				),
			},
		},
	})
}

func TestAccRedfishVirtualMedia_basic_cd_https(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPrepareVMediaSlots(creds) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, os.Getenv("TF_TESTING_VMEDIA_CD_PATH_HTTPS"), "HTTPS",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_name, "image", os.Getenv("TF_TESTING_VMEDIA_CD_PATH_HTTPS")),
					resource.TestCheckResourceAttr(resource_name, "inserted", "true"),
				),
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
					creds, os.Getenv("TF_TESTING_VMEDIA_HD_PATH_NFS"), "NFS",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_name, "image", os.Getenv("TF_TESTING_VMEDIA_HD_PATH_NFS")),
					resource.TestCheckResourceAttr(resource_name, "inserted", "true"),
				),
			},
			{
				ResourceName:      resource_name,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(d *terraform.State) (string, error) {
					return getVMediaImportHdConfiguration(creds)
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
					creds, "https://10.172.181.125:8006/gauge/vmedia/Cd!123.iso2", "HTTPS",
				),
				ExpectError: regexp.MustCompile("Image type format is not supported"),
			},
			{
				Config: testAccRedfishResourceVirtualMediaConfig(
					creds, "https://10.172.181.125:8006/gauge/vmedia/Hd!123.ima", "HTTPS",
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
