package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const (
	PERSISTENT_BOOT_ORDER_KEY = "PersistentBootConfigOrder"
	BIOS_SETTINGS_ENDPOINT    = "/redfish/v1/Systems/0/Bios/Settings"
)

func waitTillBiosSettingsApplied(ctx context.Context, service *gofish.Service, timeout int64, resetType redfish.ResetType) (diags diag.Diagnostics) {
	poweredOn, err := isPoweredOn(service)
	if err != nil {
		diags.AddError("Could not retrieve current power state", err.Error())
		return diags
	}

	var logMsg string
	logMsg = fmt.Sprintf("Process will wait with %d seconds timeout to finish", timeout)
	tflog.Info(ctx, logMsg)

	startTime := time.Now().Unix()

	if !poweredOn {
		err = changePowerState(service, true, timeout)
	} else {
		err = resetHost(service, resetType, timeout)
	}

	// Due to BIOS setting change it might happen that host will be powered off after
	// BIOS POST phase, so to not break the process the error must be omitted
	if err.Error() != "BIOS exited POST but host powered off" {
		diags.AddError("Host could not be powered on to finish BIOS settings", err.Error())
		return diags
	}

	if time.Now().Unix()-startTime > timeout {
		diags.AddError("Job timeout exceeded after reset/power on while operation has not finished", "Terminate")
		return diags
	}

	for {
		numberOfKeysInMap, diags := getBiosSettingsFutureAttributesNumber(service)
		if diags.HasError() {
			return diags
		}

		/*
		   At the moment the only way to check if process is finished is to check
		   Attributes parameter of /Bios/Settings. In case of parameters not yet applied,
		   it will contain only these which are planned to be applied. After process complete,
		   it will contain all writable properties. It's not best mechanism, but the only one known as of now
		*/
		if numberOfKeysInMap > 5 {
			var logMsg string = fmt.Sprintf("Number of keys %d", numberOfKeysInMap)
			tflog.Info(ctx, logMsg)
			break
		}

		time.Sleep(2 * time.Second)
		if time.Now().Unix()-startTime > timeout {
			diags.AddError("Job timeout exceeded while operation has not finished", "Terminate")
			return diags
		}
	}

	return diags
}
