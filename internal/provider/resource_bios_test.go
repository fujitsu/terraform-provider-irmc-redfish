package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const bios_name = "irmc-redfish_bios.bios"

func TestAccRedfishBios(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { testChangePowerHostState(creds, true) },
				Config: testAccRedfishResourceBiosConfig_correctAttributes(
					creds, "ForceRestart",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(bios_name, "id", "/redfish/v1/Systems/0/Bios/Settings"),
					resource.TestCheckResourceAttr(bios_name, "system_reset_type", "ForceRestart"),
				),
			},
		},
	})
}

func TestAccRedfishBios_negative(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRedfishResourceBiosConfig_wrongAttribute(creds, "ForceRestart"),
				ExpectError: regexp.MustCompile("Attribute 'XXX' is not supported by the system"),
			},
			{
				Config:      testAccRedfishResourceBiosConfig_notSupportedBootSources(creds, "ForceRestart"),
				ExpectError: regexp.MustCompile("Attribute 'BootSources' is not supported by the resource"),
			},
		},
	})
}

func testAccRedfishResourceBiosConfig_correctAttributes(testingInfo TestingServerCredentials, reset_type string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_bios" "bios" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "AssetTag": "TestAssetTag"
        }
        system_reset_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		reset_type,
	)
}

func testAccRedfishResourceBiosConfig_wrongAttribute(testingInfo TestingServerCredentials, reset_type string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_bios" "bios" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "XXX": "YYY"
        }
        system_reset_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		reset_type,
	)
}

func testAccRedfishResourceBiosConfig_notSupportedBootSources(testingInfo TestingServerCredentials, reset_type string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_bios" "bios" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "BootSources": "YYY"
        }
        system_reset_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		reset_type,
	)
}
