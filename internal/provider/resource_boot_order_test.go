package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const bo_name = "irmc-redfish_boot_order.bo"

func TestAccRedfishBootOrder_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { testChangePowerHostState(creds, true) },
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST"), "ForceRestart",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(bo_name, "id", "/redfish/v1/Systems/0/Bios/Settings"),
					resource.TestCheckResourceAttr(bo_name, "system_reset_type", "ForceRestart"),
				),
			},
			{
				PreConfig: func() { testChangePowerHostState(creds, false) },
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST_OPPOSITE"), "ForceRestart",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(bo_name, "id", "/redfish/v1/Systems/0/Bios/Settings"),
					resource.TestCheckResourceAttr(bo_name, "system_reset_type", "ForceRestart"),
				),
			},
		},
	})
}

func TestAccRedfishBootOrder_negative_wrongBootOrder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST_TOO_SHORT"), "ForceRestart",
				),
				ExpectError: regexp.MustCompile("Planned boot order has different lenght than currently configured boot order"),
			},
			{
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST_DUPLICATED"), "ForceRestart",
				),
				ExpectError: regexp.MustCompile("Planned boot order does not contain all available boot options"),
			},
			{
				Config: testAccRedfishResourceBootOrderConfig(
					creds, os.Getenv("TF_TESTING_BOOT_ORDER_LIST_WRONG_BOOT_ENTRY"), "ForceRestart",
				),
				ExpectError: regexp.MustCompile("Planned changes for boot order did not pass validation"),
			},
		},
	})
}

func testAccRedfishResourceBootOrderConfig(testingInfo TestingServerCredentials,
	boot_order string,
	reset_type string,
) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_boot_order" "bo" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        boot_order = %s
        system_reset_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		boot_order,
		reset_type,
	)
}
