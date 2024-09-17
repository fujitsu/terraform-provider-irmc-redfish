package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	resource_boot_source_override = "irmc-redfish_boot_source_override.bso"
)

func TestAccRedfishBootSourceOverride_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceBootSourceOverrideConfig(creds, "Cd", "Once", "PowerCycle"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_boot_source_override, "boot_source_override_target", "Cd"),
					resource.TestCheckResourceAttr(resource_boot_source_override, "boot_source_override_enabled", "Once"),
				),
			},
			{
				Config: testAccRedfishResourceBootSourceOverrideConfig(creds, "Hdd", "Continues", "ForceRestart"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resource_boot_source_override, "boot_source_override_target", "Hdd"),
					resource.TestCheckResourceAttr(resource_boot_source_override, "boot_source_override_enabled", "Continues"),
				),
			},
		},
	})
}

func testAccRedfishResourceBootSourceOverrideConfig(testingInfo TestingServerCredentials,
	overrideTarget string,
	overrideEnabled string,
	resetType string,
) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_boot_source_override" "bso" {
	  
		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        boot_source_override_target = "%s"
		boot_source_override_enabled = "%s"
		system_reset_type = "%s"
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		overrideTarget,
		overrideEnabled,
		resetType,
	)
}
