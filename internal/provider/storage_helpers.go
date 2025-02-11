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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

func getSystemStorageFromSerialNumber(service *gofish.Service, serial string) (*redfish.Storage, error) {
	system, err := GetSystemResource(service)
	if err != nil {
		return nil, err
	}

	list_of_storage_controllers, err := system.Storage()
	if err != nil {
		return nil, err
	}

	for _, storage := range list_of_storage_controllers {
		if storage.StorageControllers[0].SerialNumber == serial {
			return storage, nil
		}
	}

	return nil, fmt.Errorf("storage controller represented by serial has not been found on list of controllers for the target system")
}

type storageControllerOem struct {
	BiosContinueOnError       string `json:"BIOSContinueOnError,omitempty"`
	BiosStatusEnabled         *bool  `json:"BIOSStatus,omitempty"`
	PatrolRead                string `json:"PatrolRead,omitempty"`
	PatrolReadRatePercent     *int64 `json:"PatrolReadRate,omitempty"`
	PatrolReadRecoverySupport *bool  `json:"PatrolReadRecoverySupport,omitempty"`
	BGIRate                   *int64 `json:"BGIRate,omitempty"`
	MDCRate                   *int64 `json:"MDCRate,omitempty"`
	RebuildRate               *int64 `json:"RebuildRate,omitempty"`
	MigrationRate             *int64 `json:"MigrationRate,omitempty"`

	SpinupDelay               *int64 `json:"SpinupDelaySec,omitempty"`
	SpindownDelay             *int64 `json:"SpindownDelayMin,omitempty"`
	SpindownUnconfiguredDrive *bool  `json:"SpindownUnconfiguredDrive,omitempty"`
	SpindownHotspare          *bool  `json:"SpindownHotspare,omitempty"`
	MDCScheduleMode           string `json:"MDCScheduleMode,omitempty"`
	MDCAbortOnError           *bool  `json:"MDCAbortOnError,omitempty"`
	CoercionMode              string `json:"CoercionMode,omitempty"`
	AutoRebuild               *bool  `json:"AutoRebuildSupport,omitempty"`
	/*
		CopybackSupport                bool   `json:"CopybackSupport,omitempty"`
		CopybackOnSmartErrorSupport    bool   `json:"CopybackOnSMARTErrSupport,omitempty"`
		CopybackOnSSDSmartErrorSupport bool   `json:"CopybackOnSSDSMARTErrSupport,omitempty"`
	*/
}

type StorageControllerFujitsuOem struct {
	Ts_fujitsu storageControllerOem `json:"ts_fujitsu"`
}

type StorageController_Fujitsu struct {
	Oem StorageControllerFujitsuOem
}

type Storage_Fujitsu struct {
	StorageControllers []StorageController_Fujitsu
}

func convertPlanToPayload(plan models.StorageResourceModel) (any, bool) {
	var payload Storage_Fujitsu
	var storageController StorageController_Fujitsu
	anyValueIntoPlan := false

	if !plan.BGIRate.IsNull() && !plan.BGIRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.BGIRate = new(int64)
		*storageController.Oem.Ts_fujitsu.BGIRate = plan.BGIRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.BGIRate = nil
	}

	if !plan.MDCRate.IsNull() && !plan.MDCRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.MDCRate = new(int64)
		*storageController.Oem.Ts_fujitsu.MDCRate = plan.MDCRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.MDCRate = nil
	}

	if !plan.BiosStatusEnabled.IsNull() && !plan.BiosStatusEnabled.IsUnknown() {
		storageController.Oem.Ts_fujitsu.BiosStatusEnabled = new(bool)
		*storageController.Oem.Ts_fujitsu.BiosStatusEnabled = plan.BiosStatusEnabled.ValueBool()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.BiosStatusEnabled = nil
	}

	if !plan.BiosContinueOnError.IsNull() && !plan.BiosContinueOnError.IsUnknown() {
		storageController.Oem.Ts_fujitsu.BiosContinueOnError = plan.BiosContinueOnError.ValueString()
		anyValueIntoPlan = true
	}

	if !plan.PatrolRead.IsNull() && !plan.PatrolRead.IsUnknown() {
		storageController.Oem.Ts_fujitsu.PatrolRead = plan.PatrolRead.ValueString()
		anyValueIntoPlan = true
	}

	if !plan.PatrolReadRate.IsNull() && !plan.PatrolReadRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.PatrolReadRatePercent = new(int64)
		*storageController.Oem.Ts_fujitsu.PatrolReadRatePercent = plan.PatrolReadRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.PatrolReadRatePercent = nil
	}

	if !plan.PatrolReadRecoverySupport.IsNull() && !plan.PatrolReadRecoverySupport.IsUnknown() {
		storageController.Oem.Ts_fujitsu.PatrolReadRecoverySupport = new(bool)
		*storageController.Oem.Ts_fujitsu.PatrolReadRecoverySupport = plan.PatrolReadRecoverySupport.ValueBool()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.PatrolReadRecoverySupport = nil
	}

	if !plan.RebuildRate.IsNull() && !plan.RebuildRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.RebuildRate = new(int64)
		*storageController.Oem.Ts_fujitsu.RebuildRate = plan.RebuildRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.RebuildRate = nil
	}

	if !plan.MigrationRate.IsNull() && !plan.MigrationRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.MigrationRate = new(int64)
		*storageController.Oem.Ts_fujitsu.MigrationRate = plan.MigrationRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.MigrationRate = nil
	}

	if !plan.SpindownDelay.IsNull() && !plan.SpindownDelay.IsUnknown() {
		storageController.Oem.Ts_fujitsu.SpindownDelay = new(int64)
		*storageController.Oem.Ts_fujitsu.SpindownDelay = plan.SpindownDelay.ValueInt64()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.SpindownDelay = nil
	}

	if !plan.SpinupDelay.IsNull() && !plan.SpinupDelay.IsUnknown() {
		storageController.Oem.Ts_fujitsu.SpinupDelay = new(int64)
		*storageController.Oem.Ts_fujitsu.SpinupDelay = plan.SpinupDelay.ValueInt64()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.SpinupDelay = nil
	}

	if !plan.SpindownUnconfDrive.IsNull() && !plan.SpindownUnconfDrive.IsUnknown() {
		storageController.Oem.Ts_fujitsu.SpindownUnconfiguredDrive = new(bool)
		*storageController.Oem.Ts_fujitsu.SpindownUnconfiguredDrive = plan.SpindownUnconfDrive.ValueBool()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.SpindownUnconfiguredDrive = nil
	}

	if !plan.SpindownHotspare.IsNull() && !plan.SpindownHotspare.IsUnknown() {
		storageController.Oem.Ts_fujitsu.SpindownHotspare = new(bool)
		*storageController.Oem.Ts_fujitsu.SpindownHotspare = plan.SpindownHotspare.ValueBool()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.SpindownHotspare = nil
	}

	if !plan.MDCScheduleMode.IsNull() && !plan.MDCScheduleMode.IsUnknown() {
		storageController.Oem.Ts_fujitsu.MDCScheduleMode = plan.MDCScheduleMode.ValueString()
		anyValueIntoPlan = true
	}

	if !plan.MDCAbortOnError.IsNull() && !plan.MDCAbortOnError.IsUnknown() {
		storageController.Oem.Ts_fujitsu.MDCAbortOnError = new(bool)
		*storageController.Oem.Ts_fujitsu.MDCAbortOnError = plan.MDCAbortOnError.ValueBool()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.MDCAbortOnError = nil
	}

	if !plan.CoercionMode.IsNull() && !plan.CoercionMode.IsUnknown() {
		storageController.Oem.Ts_fujitsu.CoercionMode = plan.CoercionMode.ValueString()
		anyValueIntoPlan = true
	}
	/*
	   	if !plan.CopybackSupport.IsNull() && !plan.CopybackSupport.IsUnknown() {
	   		*storageController.Oem.Ts_fujitsu.CopybackSupport = plan.CopybackSupport.ValueBool()
	   	} else {
	   		storageController.Oem.Ts_fujitsu.CopybackSupport = nil
	    }

	   	if !plan.CopybackOnSmartErrorSupport.IsNull() && !plan.CopybackOnSmartErrorSupport.IsUnknown() {
	   		*storageController.Oem.Ts_fujitsu.CopybackOnSmartErrorSupport = plan.CopybackOnSmartErrorSupport.ValueBool()
	   	} else {
	   		storageController.Oem.Ts_fujitsu.CopybackOnSmartErrorSupport = nil
	       }

	   	if !plan.CopybackOnSSDSmartErrorSupport.IsNull() && !plan.CopybackOnSSDSmartErrorSupport.IsUnknown() {
	   		*storageController.Oem.Ts_fujitsu.CopybackOnSSDSmartErrorSupport = plan.CopybackOnSSDSmartErrorSupport.ValueBool()
	   	} else {
	   		storageController.Oem.Ts_fujitsu.CopybackOnSSDSmartErrorSupport = nil
	       }
	*/
	if !plan.AutoRebuild.IsNull() && !plan.AutoRebuild.IsUnknown() {
		storageController.Oem.Ts_fujitsu.AutoRebuild = new(bool)
		*storageController.Oem.Ts_fujitsu.AutoRebuild = plan.AutoRebuild.ValueBool()
		anyValueIntoPlan = true
	} else {
		storageController.Oem.Ts_fujitsu.AutoRebuild = nil
	}

	payload.StorageControllers = append(payload.StorageControllers, storageController)
	return payload, anyValueIntoPlan
}

type ExtendedInfoMsg struct {
	MessageId string `json:"MessageId"`
	Message   string `json:"Message"`
}

type StoragePatchResponse struct {
	ExtendedInfo []ExtendedInfoMsg `json:"@Message.ExtendedInfo"`
}

func patchStorageEndpoint(ctx context.Context, service *gofish.Service, endpoint string, payload any) (taskLocation string, err error) {
	tflog.Info(ctx, "Payload will be PATCHed to controller", map[string]interface{}{
		"storage endpoint": endpoint,
		"payload":          payload,
	})

	resp, err := service.GetClient().Patch(endpoint, payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PATCH request on '%s' finished with not expected status '%d'", endpoint, resp.StatusCode)
	}

	// Request might be accepted but some properties will not be successfully validated and it should be reported to terraform
	out, err := io.ReadAll(resp.Body)
	var respStruct StoragePatchResponse

	err = json.Unmarshal(out, &respStruct)
	if err != nil {
		return "", err
	}

	if len(respStruct.ExtendedInfo) > 0 {
		for _, v := range respStruct.ExtendedInfo {
			tflog.Warn(ctx, "Request responded with non-empty ExtendedMessageInfo", map[string]interface{}{
				"MessageId": v.MessageId,
				"Message":   v.Message,
			})
		}
	}

	return "", err
}

func checkAppliedSettingsFromPlan(ctx context.Context, plan models.StorageResourceModel, current Storage_Fujitsu) bool {
	status := true

	if !plan.BiosContinueOnError.IsNull() && !plan.BiosContinueOnError.IsUnknown() {
		if plan.BiosContinueOnError.ValueString() != current.StorageControllers[0].Oem.Ts_fujitsu.BiosContinueOnError {
			status = false
			tflog.Info(ctx, "Value for property BIOSContinueOnError has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BiosContinueOnError.ValueString(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.BiosContinueOnError,
			})
		}
	}

	if !plan.BiosStatusEnabled.IsNull() && !plan.BiosStatusEnabled.IsUnknown() {
		if plan.BiosStatusEnabled.ValueBool() != *current.StorageControllers[0].Oem.Ts_fujitsu.BiosStatusEnabled {
			status = false
			tflog.Info(ctx, "Value for property BIOSStatus has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BiosStatusEnabled.ValueBool(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.BiosStatusEnabled,
			})
		}
	}

	if !plan.PatrolRead.IsNull() && !plan.PatrolRead.IsUnknown() {
		if plan.PatrolRead.ValueString() != current.StorageControllers[0].Oem.Ts_fujitsu.PatrolRead {
			status = false
			tflog.Info(ctx, "Value for property PatrolRead has not yet reached planned value", map[string]interface{}{
				"plan":     plan.PatrolRead.ValueString(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.PatrolRead,
			})
		}
	}

	if !plan.PatrolReadRate.IsNull() && !plan.PatrolReadRate.IsUnknown() {
		if plan.PatrolReadRate.ValueInt64() != *current.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRatePercent {
			status = false
			tflog.Info(ctx, "Value for property PatrolReadRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.PatrolReadRate.ValueInt64(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRatePercent,
			})
		}
	}

	if !plan.PatrolReadRecoverySupport.IsNull() && !plan.PatrolReadRecoverySupport.IsUnknown() {
		if plan.PatrolReadRecoverySupport.ValueBool() != *current.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRecoverySupport {
			status = false
			tflog.Info(ctx, "Value for property PatrolReadRecoverySupport has not yet reached planned value", map[string]interface{}{
				"plan":     plan.PatrolReadRecoverySupport.ValueBool(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRecoverySupport,
			})
		}
	}

	if !plan.BGIRate.IsNull() && !plan.BGIRate.IsUnknown() {
		if plan.BGIRate.ValueInt64() != *current.StorageControllers[0].Oem.Ts_fujitsu.BGIRate {
			status = false
			tflog.Info(ctx, "Value for property BGIRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BGIRate.ValueInt64(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.BGIRate,
			})
		}
	}

	if !plan.MDCRate.IsNull() && !plan.MDCRate.IsUnknown() {
		if plan.MDCRate.ValueInt64() != *current.StorageControllers[0].Oem.Ts_fujitsu.MDCRate {
			status = false
			tflog.Info(ctx, "Value for property MDCRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MDCRate.ValueInt64(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.MDCRate,
			})
		}
	}

	if !plan.RebuildRate.IsNull() && !plan.RebuildRate.IsUnknown() {
		if plan.RebuildRate.ValueInt64() != *current.StorageControllers[0].Oem.Ts_fujitsu.RebuildRate {
			status = false
			tflog.Info(ctx, "Value for property RebuildRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.RebuildRate.ValueInt64(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.RebuildRate,
			})
		}
	}

	if !plan.MigrationRate.IsNull() && !plan.MigrationRate.IsUnknown() {
		if plan.MigrationRate.ValueInt64() != *current.StorageControllers[0].Oem.Ts_fujitsu.MigrationRate {
			status = false
			tflog.Info(ctx, "Value for property MigrationRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MigrationRate.ValueInt64(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.MigrationRate,
			})
		}
	}

	if !plan.SpindownDelay.IsNull() && !plan.SpindownDelay.IsUnknown() {
		if plan.SpindownDelay.ValueInt64() != *current.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay {
			status = false
			tflog.Info(ctx, "Value for property SpindownDelay has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MigrationRate.ValueInt64(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay,
			})
		}
	}

	if !plan.SpinupDelay.IsNull() && !plan.SpinupDelay.IsUnknown() {
		if plan.SpinupDelay.ValueInt64() != *current.StorageControllers[0].Oem.Ts_fujitsu.SpinupDelay {
			status = false
			tflog.Info(ctx, "Value for property SpinupDelay has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpinupDelay.ValueInt64(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.SpinupDelay,
			})
		}
	}

	if !plan.SpindownUnconfDrive.IsNull() && !plan.SpindownUnconfDrive.IsUnknown() {
		if plan.SpindownUnconfDrive.ValueBool() != *current.StorageControllers[0].Oem.Ts_fujitsu.SpindownUnconfiguredDrive {
			status = false
			tflog.Info(ctx, "Value for property SpindownUnconfiguredDrive has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpindownUnconfDrive.ValueBool(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.SpindownUnconfiguredDrive,
			})
		}
	}

	if !plan.SpindownHotspare.IsNull() && !plan.SpindownHotspare.IsUnknown() {
		if plan.SpindownHotspare.ValueBool() != *current.StorageControllers[0].Oem.Ts_fujitsu.SpindownHotspare {
			status = false
			tflog.Info(ctx, "Value for property SpindownHotspare has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpindownHotspare.ValueBool(),
				"reported": *current.StorageControllers[0].Oem.Ts_fujitsu.SpindownHotspare,
			})
		}
	}

	if !plan.MDCScheduleMode.IsNull() && !plan.MDCScheduleMode.IsUnknown() {
		if plan.MDCScheduleMode.ValueString() != current.StorageControllers[0].Oem.Ts_fujitsu.MDCScheduleMode {
			status = false
			tflog.Info(ctx, "Value for property MDCScheduleMode has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MDCScheduleMode.ValueString(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.MDCScheduleMode,
			})
		}
	}

	if !plan.MDCAbortOnError.IsNull() && !plan.MDCAbortOnError.IsUnknown() {
		if plan.MDCAbortOnError.ValueBool() != *current.StorageControllers[0].Oem.Ts_fujitsu.MDCAbortOnError {
			status = false
			tflog.Info(ctx, "Value for property MDCAbortOnError has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MDCAbortOnError.ValueBool(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.MDCAbortOnError,
			})
		}
	}

	if !plan.CoercionMode.IsNull() && !plan.CoercionMode.IsUnknown() {
		if plan.CoercionMode.ValueString() != current.StorageControllers[0].Oem.Ts_fujitsu.CoercionMode {
			status = false
			tflog.Info(ctx, "Value for property CoercionMode has not yet reached planned value", map[string]interface{}{
				"plan":     plan.CoercionMode.ValueString(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.CoercionMode,
			})
		}
	}

	/*
		if !plan.CopybackSupport.IsNull() && !plan.CopybackSupport.IsUnknown() {
			if plan.CopybackSupport.ValueBool() != current.StorageControllers[0].Oem.Ts_fujitsu.CopybackSupport {
				status = false
				tflog.Info(ctx, "Value for property CopybackSupport has not yet reached planned value", map[string]interface{}{
					"plan":     plan.CopybackSupport.ValueBool(),
					"reported": current.StorageControllers[0].Oem.Ts_fujitsu.CopybackSupport,
				})
			}
		}

		if !plan.CopybackOnSmartErrorSupport.IsNull() && !plan.CopybackOnSmartErrorSupport.IsUnknown() {
			if plan.CopybackOnSmartErrorSupport.ValueBool() != current.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSmartErrorSupport {
				status = false
				tflog.Info(ctx, "Value for property CopybackOnSmartErrorSupport has not yet reached planned value", map[string]interface{}{
					"plan":     plan.CopybackOnSmartErrorSupport.ValueBool(),
					"reported": current.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSmartErrorSupport,
				})
			}
		}

		if !plan.CopybackOnSSDSmartErrorSupport.IsNull() && !plan.CopybackOnSSDSmartErrorSupport.IsUnknown() {
			if plan.CopybackOnSSDSmartErrorSupport.ValueBool() != current.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSSDSmartErrorSupport {
				status = false
				tflog.Info(ctx, "Value for property CopybackOnSSDSmartErrorSupport has not yet reached planned value", map[string]interface{}{
					"plan":     plan.CopybackOnSSDSmartErrorSupport.ValueBool(),
					"reported": current.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSSDSmartErrorSupport,
				})
			}
		}
	*/
	if !plan.AutoRebuild.IsNull() && !plan.AutoRebuild.IsUnknown() {
		if plan.AutoRebuild.ValueBool() != *current.StorageControllers[0].Oem.Ts_fujitsu.AutoRebuild {
			status = false
			tflog.Info(ctx, "Value for property AutoRebuild has not yet reached planned value", map[string]interface{}{
				"plan":     plan.AutoRebuild.ValueBool(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.AutoRebuild,
			})
		}
	}

	if status {
		tflog.Info(ctx, "All values from plan has been successfully applied")
	} else {
		tflog.Trace(ctx, "NOT all values from plan has been already applied, need to retry check")
	}

	return status
}

func checkIfPlannedStorageChangesSuccessfullyApplied(ctx context.Context, service *gofish.Service, plan models.StorageResourceModel) bool {
	var storageResource Storage_Fujitsu
	_, err := readStorageControllerSettings(service, plan.StorageControllerSN.ValueString(), &storageResource)
	if err != nil {
		tflog.Error(ctx, err.Error())
		return false
	}

	return checkAppliedSettingsFromPlan(ctx, plan, storageResource)
}

func waitUntilStorageChangesApplied(ctx context.Context, service *gofish.Service, taskLocation string, plan models.StorageResourceModel, startTime int64, timeout int64) (status bool, err error) {
	for {
		if len(taskLocation) == 0 {
			if checkIfPlannedStorageChangesSuccessfullyApplied(ctx, service, plan) {
				return true, err
			}
		}
		// TODO: no support for task approach

		if time.Now().Unix()-startTime > timeout {
			return false, fmt.Errorf("timeout of %d s has been reached", timeout)
		}

		time.Sleep(5 * time.Second)
	}
}

func applyStorageControllerProperties(ctx context.Context, service *gofish.Service, plan *models.StorageResourceModel) (diags diag.Diagnostics) {
	storage, err := getSystemStorageFromSerialNumber(service, plan.StorageControllerSN.ValueString())
	if err != nil {
		diags.AddError("Requested storage serial does not match to any installed controller serial.", err.Error())
		return diags
	}

	tflog.Info(ctx, "Serial number", map[string]interface{}{
		"serial": plan.StorageControllerSN.ValueString(),
	})

	payload, anyValue := convertPlanToPayload(*plan)

	if !anyValue {
		diags.AddError("Payload created out of defined plan will be empty.",
			"Declare at least one property which is expected to be set")
		return diags
	}

	startTime := time.Now().Unix()
	timeout := plan.JobTimeout.ValueInt64()
	taskLocation, err := patchStorageEndpoint(ctx, service, storage.ODataID, payload)
	if err != nil {
		diags.AddError("Error during PATCH to storage controller.", err.Error())
		return diags
	}

	if time.Now().Unix()-startTime > timeout {
		diags.AddError("Error while waiting for resource update.", fmt.Sprintf("Timeout of %d s has been reached", timeout))
		return diags
	}

	_, err = waitUntilStorageChangesApplied(ctx, service, taskLocation, *plan, startTime, timeout)
	if err != nil {
		diags.AddError("Error while waiting for resource update.", err.Error())
		return diags
	}

	plan.Id = types.StringValue(storage.ODataID)
	return diags
}

func getStorageResource(service *gofish.Service, endpoint string) (out []byte, err error) {
	resp, err := service.GetClient().Get(endpoint)
	if err != nil {
		return out, err
	}

	if resp.StatusCode == http.StatusOK {
		out, err = io.ReadAll(resp.Body)
	} else {
		err = fmt.Errorf("GET on '%s' returned unexpected status '%d'", endpoint, resp.StatusCode)
	}

	return out, err
}

func getParsedStorageResource(service *gofish.Service, endpoint string, config *Storage_Fujitsu) error {
	body, err := getStorageResource(service, endpoint)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, config)
}

func copyStorageConfigIntoModel(storageConfig Storage_Fujitsu, state *models.StorageSettings) {
	state.BiosContinueOnError = types.StringValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.BiosContinueOnError)
	state.PatrolRead = types.StringValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.PatrolRead)
	state.MDCScheduleMode = types.StringValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MDCScheduleMode)
	state.CoercionMode = types.StringValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CoercionMode)

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.BiosStatusEnabled != nil {
		state.BiosStatusEnabled = types.BoolValue(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.BiosStatusEnabled)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRatePercent != nil {
		state.PatrolReadRate = types.Int64Value(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRatePercent)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRecoverySupport != nil {
		state.PatrolReadRecoverySupport = types.BoolValue(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRecoverySupport)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.BGIRate != nil {
		state.BGIRate = types.Int64Value(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.BGIRate)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MDCRate != nil {
		state.MDCRate = types.Int64Value(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MDCRate)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.RebuildRate != nil {
		state.RebuildRate = types.Int64Value(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.RebuildRate)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MigrationRate != nil {
		state.MigrationRate = types.Int64Value(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MigrationRate)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay != nil {
		state.SpindownDelay = types.Int64Value(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay != nil {
		state.SpinupDelay = types.Int64Value(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownUnconfiguredDrive != nil {
		state.SpindownUnconfDrive = types.BoolValue(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownUnconfiguredDrive)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownHotspare != nil {
		state.SpindownHotspare = types.BoolValue(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownHotspare)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MDCAbortOnError != nil {
		state.MDCAbortOnError = types.BoolValue(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MDCAbortOnError)
	}

	if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.AutoRebuild != nil {
		state.AutoRebuild = types.BoolValue(*storageConfig.StorageControllers[0].Oem.Ts_fujitsu.AutoRebuild)
	}

	/*
				if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackSupport != nil {
		    		state.CopybackSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackSupport)
		        }

				if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSmartErrorSupport != nil {
		    		state.CopybackOnSmartErrorSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSmartErrorSupport)
		        }

				if storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSSDSmartErrorSupport != nil {
		    		state.CopybackOnSSDSmartErrorSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSSDSmartErrorSupport)
		        }
	*/
}

func readStorageControllerSettings(service *gofish.Service, serialNumber string, storageResource *Storage_Fujitsu) (odataid string, err error) {
	storage, err := getSystemStorageFromSerialNumber(service, serialNumber)
	if err != nil {
		return "", err
	}

	err = getParsedStorageResource(service, storage.ODataID, storageResource)
	if err != nil {
		return "", err
	}

	return storage.ODataID, nil
}

func readStorageControllerSettingsToState(service *gofish.Service, state *models.StorageSettings) (odataid string, diags diag.Diagnostics) {
	var storageResource Storage_Fujitsu
	odataid, err := readStorageControllerSettings(service, state.StorageControllerSN.ValueString(), &storageResource)
	if err != nil {
		diags.AddError("Could not obtain storage resource settings", err.Error())
		return odataid, diags
	}

	copyStorageConfigIntoModel(storageResource, state)
	return odataid, diags
}
