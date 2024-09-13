package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRedfishVirtualMedia_fetch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishDatasourceVirtualMediaConfig(creds),
			},
		},
	})
}

func testAccRedfishDatasourceVirtualMediaConfig(testingInfo TestingServerCredentials) string {
	return fmt.Sprintf(`
	data "irmc-redfish_virtual_media" "vm" {
	  
		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}
	  }
	  
	  output "virtual_media" {
		 value = data.irmc-redfish_virtual_media.vm
		 sensitive = true
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
	)
}
