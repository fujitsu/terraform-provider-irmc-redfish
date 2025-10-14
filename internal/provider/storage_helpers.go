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
	OemFujitsu *storageControllerOem `json:"ts_fujitsu,omitempty"`
	OemFsas    *storageControllerOem `json:"Fsas,omitempty"`
}

type StorageController_Fujitsu struct {
	Oem StorageControllerFujitsuOem
}

type Storage_Fujitsu struct {
	StorageControllers []StorageController_Fujitsu
}

func getOemStorage(oem StorageControllerFujitsuOem) storageControllerOem {
	if oem.OemFujitsu != nil {
		return *oem.OemFujitsu
	}

	return *oem.OemFsas
}

func convertPlanToPayload(isFsas bool, plan models.StorageResourceModel) (any, bool) {
	var storageController StorageController_Fujitsu
	anyValueIntoPlan := false

	var oemObject storageControllerOem
	oem := storageController.Oem.OemFsas

	if isFsas {
		storageController.Oem.OemFsas = &oemObject
		storageController.Oem.OemFujitsu = nil
		oem = storageController.Oem.OemFsas
	} else {
		storageController.Oem.OemFujitsu = &oemObject
		storageController.Oem.OemFsas = nil
		oem = storageController.Oem.OemFujitsu
	}

	if !plan.BGIRate.IsNull() && !plan.BGIRate.IsUnknown() {
		(*oem).BGIRate = new(int64)
		*(*oem).BGIRate = plan.BGIRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		(*oem).BGIRate = nil
	}

	if !plan.MDCRate.IsNull() && !plan.MDCRate.IsUnknown() {
		(*oem).MDCRate = new(int64)
		*(*oem).MDCRate = plan.MDCRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		(*oem).MDCRate = nil
	}

	if !plan.BiosStatusEnabled.IsNull() && !plan.BiosStatusEnabled.IsUnknown() {
		(*oem).BiosStatusEnabled = new(bool)
		*(*oem).BiosStatusEnabled = plan.BiosStatusEnabled.ValueBool()
		anyValueIntoPlan = true
	} else {
		(*oem).BiosStatusEnabled = nil
	}

	if !plan.BiosContinueOnError.IsNull() && !plan.BiosContinueOnError.IsUnknown() {
		(*oem).BiosContinueOnError = plan.BiosContinueOnError.ValueString()
		anyValueIntoPlan = true
	}

	if !plan.PatrolRead.IsNull() && !plan.PatrolRead.IsUnknown() {
		(*oem).PatrolRead = plan.PatrolRead.ValueString()
		anyValueIntoPlan = true
	}

	if !plan.PatrolReadRate.IsNull() && !plan.PatrolReadRate.IsUnknown() {
		(*oem).PatrolReadRatePercent = new(int64)
		*(*oem).PatrolReadRatePercent = plan.PatrolReadRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		(*oem).PatrolReadRatePercent = nil
	}

	if !plan.PatrolReadRecoverySupport.IsNull() && !plan.PatrolReadRecoverySupport.IsUnknown() {
		(*oem).PatrolReadRecoverySupport = new(bool)
		*(*oem).PatrolReadRecoverySupport = plan.PatrolReadRecoverySupport.ValueBool()
		anyValueIntoPlan = true
	} else {
		(*oem).PatrolReadRecoverySupport = nil
	}

	if !plan.RebuildRate.IsNull() && !plan.RebuildRate.IsUnknown() {
		(*oem).RebuildRate = new(int64)
		*(*oem).RebuildRate = plan.RebuildRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		(*oem).RebuildRate = nil
	}

	if !plan.MigrationRate.IsNull() && !plan.MigrationRate.IsUnknown() {
		(*oem).MigrationRate = new(int64)
		*(*oem).MigrationRate = plan.MigrationRate.ValueInt64()
		anyValueIntoPlan = true
	} else {
		(*oem).MigrationRate = nil
	}

	if !plan.SpindownDelay.IsNull() && !plan.SpindownDelay.IsUnknown() {
		(*oem).SpindownDelay = new(int64)
		*(*oem).SpindownDelay = plan.SpindownDelay.ValueInt64()
		anyValueIntoPlan = true
	} else {
		(*oem).SpindownDelay = nil
	}

	if !plan.SpinupDelay.IsNull() && !plan.SpinupDelay.IsUnknown() {
		(*oem).SpinupDelay = new(int64)
		*(*oem).SpinupDelay = plan.SpinupDelay.ValueInt64()
		anyValueIntoPlan = true
	} else {
		(*oem).SpinupDelay = nil
	}

	if !plan.SpindownUnconfDrive.IsNull() && !plan.SpindownUnconfDrive.IsUnknown() {
		(*oem).SpindownUnconfiguredDrive = new(bool)
		*(*oem).SpindownUnconfiguredDrive = plan.SpindownUnconfDrive.ValueBool()
		anyValueIntoPlan = true
	} else {
		(*oem).SpindownUnconfiguredDrive = nil
	}

	if !plan.SpindownHotspare.IsNull() && !plan.SpindownHotspare.IsUnknown() {
		(*oem).SpindownHotspare = new(bool)
		*(*oem).SpindownHotspare = plan.SpindownHotspare.ValueBool()
		anyValueIntoPlan = true
	} else {
		(*oem).SpindownHotspare = nil
	}

	if !plan.MDCScheduleMode.IsNull() && !plan.MDCScheduleMode.IsUnknown() {
		(*oem).MDCScheduleMode = plan.MDCScheduleMode.ValueString()
		anyValueIntoPlan = true
	}

	if !plan.MDCAbortOnError.IsNull() && !plan.MDCAbortOnError.IsUnknown() {
		(*oem).MDCAbortOnError = new(bool)
		*(*oem).MDCAbortOnError = plan.MDCAbortOnError.ValueBool()
		anyValueIntoPlan = true
	} else {
		(*oem).MDCAbortOnError = nil
	}

	if !plan.CoercionMode.IsNull() && !plan.CoercionMode.IsUnknown() {
		(*oem).CoercionMode = plan.CoercionMode.ValueString()
		anyValueIntoPlan = true
	}

	/*
	   	if !plan.CopybackSupport.IsNull() && !plan.CopybackSupport.IsUnknown() {
	   		*oem.CopybackSupport = plan.CopybackSupport.ValueBool()
	   	} else {
	   		oem.CopybackSupport = nil
	    }

	   	if !plan.CopybackOnSmartErrorSupport.IsNull() && !plan.CopybackOnSmartErrorSupport.IsUnknown() {
	   		*oem.CopybackOnSmartErrorSupport = plan.CopybackOnSmartErrorSupport.ValueBool()
	   	} else {
	   		oem.CopybackOnSmartErrorSupport = nil
	       }

	   	if !plan.CopybackOnSSDSmartErrorSupport.IsNull() && !plan.CopybackOnSSDSmartErrorSupport.IsUnknown() {
	   		*oem.CopybackOnSSDSmartErrorSupport = plan.CopybackOnSSDSmartErrorSupport.ValueBool()
	   	} else {
	   		oem.CopybackOnSSDSmartErrorSupport = nil
	       }
	*/

	if !plan.AutoRebuild.IsNull() && !plan.AutoRebuild.IsUnknown() {
		(*oem).AutoRebuild = new(bool)
		*(*oem).AutoRebuild = plan.AutoRebuild.ValueBool()
		anyValueIntoPlan = true
	} else {
		(*oem).AutoRebuild = nil
	}

	var payload Storage_Fujitsu
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("PATCH request on '%s' finished with not expected status '%d'", endpoint, resp.StatusCode)
	}

	if resp.StatusCode == http.StatusAccepted {
		taskLocation := resp.Header.Get(HTTP_HEADER_LOCATION)
		if taskLocation == "" {
			return "", fmt.Errorf("Location header not found in response")
		}
		return taskLocation, nil
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
		if plan.BiosContinueOnError.ValueString() != getOemStorage(current.StorageControllers[0].Oem).BiosContinueOnError {
			status = false
			tflog.Info(ctx, "Value for property BIOSContinueOnError has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BiosContinueOnError.ValueString(),
				"reported": getOemStorage(current.StorageControllers[0].Oem).BiosContinueOnError,
			})
		}
	}

	if !plan.BiosStatusEnabled.IsNull() && !plan.BiosStatusEnabled.IsUnknown() {
		if plan.BiosStatusEnabled.ValueBool() != *(getOemStorage(current.StorageControllers[0].Oem).BiosStatusEnabled) {
			status = false
			tflog.Info(ctx, "Value for property BIOSStatus has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BiosStatusEnabled.ValueBool(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).BiosStatusEnabled),
			})
		}
	}

	if !plan.PatrolRead.IsNull() && !plan.PatrolRead.IsUnknown() {
		if plan.PatrolRead.ValueString() != getOemStorage(current.StorageControllers[0].Oem).PatrolRead {
			status = false
			tflog.Info(ctx, "Value for property PatrolRead has not yet reached planned value", map[string]interface{}{
				"plan":     plan.PatrolRead.ValueString(),
				"reported": getOemStorage(current.StorageControllers[0].Oem).PatrolRead,
			})
		}
	}

	if !plan.PatrolReadRate.IsNull() && !plan.PatrolReadRate.IsUnknown() {
		if plan.PatrolReadRate.ValueInt64() != *(getOemStorage(current.StorageControllers[0].Oem).PatrolReadRatePercent) {
			status = false
			tflog.Info(ctx, "Value for property PatrolReadRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.PatrolReadRate.ValueInt64(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).PatrolReadRatePercent),
			})
		}
	}

	if !plan.PatrolReadRecoverySupport.IsNull() && !plan.PatrolReadRecoverySupport.IsUnknown() {
		if plan.PatrolReadRecoverySupport.ValueBool() != *(getOemStorage(current.StorageControllers[0].Oem).PatrolReadRecoverySupport) {
			status = false
			tflog.Info(ctx, "Value for property PatrolReadRecoverySupport has not yet reached planned value", map[string]interface{}{
				"plan":     plan.PatrolReadRecoverySupport.ValueBool(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).PatrolReadRecoverySupport),
			})
		}
	}

	if !plan.BGIRate.IsNull() && !plan.BGIRate.IsUnknown() {
		if plan.BGIRate.ValueInt64() != *(getOemStorage(current.StorageControllers[0].Oem).BGIRate) {
			status = false
			tflog.Info(ctx, "Value for property BGIRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BGIRate.ValueInt64(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).BGIRate),
			})
		}
	}

	if !plan.MDCRate.IsNull() && !plan.MDCRate.IsUnknown() {
		if plan.MDCRate.ValueInt64() != *(getOemStorage(current.StorageControllers[0].Oem).MDCRate) {
			status = false
			tflog.Info(ctx, "Value for property MDCRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MDCRate.ValueInt64(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).MDCRate),
			})
		}
	}

	if !plan.RebuildRate.IsNull() && !plan.RebuildRate.IsUnknown() {
		if plan.RebuildRate.ValueInt64() != *(getOemStorage(current.StorageControllers[0].Oem).RebuildRate) {
			status = false
			tflog.Info(ctx, "Value for property RebuildRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.RebuildRate.ValueInt64(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).RebuildRate),
			})
		}
	}

	if !plan.MigrationRate.IsNull() && !plan.MigrationRate.IsUnknown() {
		if plan.MigrationRate.ValueInt64() != *(getOemStorage(current.StorageControllers[0].Oem).MigrationRate) {
			status = false
			tflog.Info(ctx, "Value for property MigrationRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MigrationRate.ValueInt64(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).MigrationRate),
			})
		}
	}

	if !plan.SpindownDelay.IsNull() && !plan.SpindownDelay.IsUnknown() {
		if plan.SpindownDelay.ValueInt64() != *(getOemStorage(current.StorageControllers[0].Oem).SpindownDelay) {
			status = false
			tflog.Info(ctx, "Value for property SpindownDelay has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MigrationRate.ValueInt64(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).SpindownDelay),
			})
		}
	}

	if !plan.SpinupDelay.IsNull() && !plan.SpinupDelay.IsUnknown() {
		if plan.SpinupDelay.ValueInt64() != *(getOemStorage(current.StorageControllers[0].Oem).SpinupDelay) {
			status = false
			tflog.Info(ctx, "Value for property SpinupDelay has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpinupDelay.ValueInt64(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).SpinupDelay),
			})
		}
	}

	if !plan.SpindownUnconfDrive.IsNull() && !plan.SpindownUnconfDrive.IsUnknown() {
		if plan.SpindownUnconfDrive.ValueBool() != *(getOemStorage(current.StorageControllers[0].Oem).SpindownUnconfiguredDrive) {
			status = false
			tflog.Info(ctx, "Value for property SpindownUnconfiguredDrive has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpindownUnconfDrive.ValueBool(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).SpindownUnconfiguredDrive),
			})
		}
	}

	if !plan.SpindownHotspare.IsNull() && !plan.SpindownHotspare.IsUnknown() {
		if plan.SpindownHotspare.ValueBool() != *(getOemStorage(current.StorageControllers[0].Oem).SpindownHotspare) {
			status = false
			tflog.Info(ctx, "Value for property SpindownHotspare has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpindownHotspare.ValueBool(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).SpindownHotspare),
			})
		}
	}

	if !plan.MDCScheduleMode.IsNull() && !plan.MDCScheduleMode.IsUnknown() {
		if plan.MDCScheduleMode.ValueString() != getOemStorage(current.StorageControllers[0].Oem).MDCScheduleMode {
			status = false
			tflog.Info(ctx, "Value for property MDCScheduleMode has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MDCScheduleMode.ValueString(),
				"reported": getOemStorage(current.StorageControllers[0].Oem).MDCScheduleMode,
			})
		}
	}

	if !plan.MDCAbortOnError.IsNull() && !plan.MDCAbortOnError.IsUnknown() {
		if plan.MDCAbortOnError.ValueBool() != *(getOemStorage(current.StorageControllers[0].Oem).MDCAbortOnError) {
			status = false
			tflog.Info(ctx, "Value for property MDCAbortOnError has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MDCAbortOnError.ValueBool(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).MDCAbortOnError),
			})
		}
	}

	if !plan.CoercionMode.IsNull() && !plan.CoercionMode.IsUnknown() {
		if plan.CoercionMode.ValueString() != getOemStorage(current.StorageControllers[0].Oem).CoercionMode {
			status = false
			tflog.Info(ctx, "Value for property CoercionMode has not yet reached planned value", map[string]interface{}{
				"plan":     plan.CoercionMode.ValueString(),
				"reported": getOemStorage(current.StorageControllers[0].Oem).CoercionMode,
			})
		}
	}

	/*
		if !plan.CopybackSupport.IsNull() && !plan.CopybackSupport.IsUnknown() {
			if plan.CopybackSupport.ValueBool() != current.StorageControllers[0].Oem.OemFujitsu.CopybackSupport {
				status = false
				tflog.Info(ctx, "Value for property CopybackSupport has not yet reached planned value", map[string]interface{}{
					"plan":     plan.CopybackSupport.ValueBool(),
					"reported": current.StorageControllers[0].Oem.OemFujitsu.CopybackSupport,
				})
			}
		}

		if !plan.CopybackOnSmartErrorSupport.IsNull() && !plan.CopybackOnSmartErrorSupport.IsUnknown() {
			if plan.CopybackOnSmartErrorSupport.ValueBool() != current.StorageControllers[0].Oem.OemFujitsu.CopybackOnSmartErrorSupport {
				status = false
				tflog.Info(ctx, "Value for property CopybackOnSmartErrorSupport has not yet reached planned value", map[string]interface{}{
					"plan":     plan.CopybackOnSmartErrorSupport.ValueBool(),
					"reported": current.StorageControllers[0].Oem.OemFujitsu.CopybackOnSmartErrorSupport,
				})
			}
		}

		if !plan.CopybackOnSSDSmartErrorSupport.IsNull() && !plan.CopybackOnSSDSmartErrorSupport.IsUnknown() {
			if plan.CopybackOnSSDSmartErrorSupport.ValueBool() != current.StorageControllers[0].Oem.OemFujitsu.CopybackOnSSDSmartErrorSupport {
				status = false
				tflog.Info(ctx, "Value for property CopybackOnSSDSmartErrorSupport has not yet reached planned value", map[string]interface{}{
					"plan":     plan.CopybackOnSSDSmartErrorSupport.ValueBool(),
					"reported": current.StorageControllers[0].Oem.OemFujitsu.CopybackOnSSDSmartErrorSupport,
				})
			}
		}
	*/
	if !plan.AutoRebuild.IsNull() && !plan.AutoRebuild.IsUnknown() {
		if plan.AutoRebuild.ValueBool() != *(getOemStorage(current.StorageControllers[0].Oem).AutoRebuild) {
			status = false
			tflog.Info(ctx, "Value for property AutoRebuild has not yet reached planned value", map[string]interface{}{
				"plan":     plan.AutoRebuild.ValueBool(),
				"reported": *(getOemStorage(current.StorageControllers[0].Oem).AutoRebuild),
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

func waitUntilStorageChangesApplied(ctx context.Context, service *gofish.Service, task_location string,
	plan models.StorageResourceModel, startTime int64, is_fsas bool, timeout int64) (diags diag.Diagnostics) {

	if len(task_location) != 0 {
		_, err := WaitForRedfishTaskEnd(ctx, service, task_location, timeout)
		if err != nil {
			diags.AddError("Task for storage controller modification reported error", err.Error())
			logs, internal_diags := FetchRedfishTaskLog(service, task_location, is_fsas)
			if logs == nil {
				diags = append(diags, internal_diags...)
			} else {
				diags.AddError("Task logs for volume creation", string(logs))
			}

			return diags
		}
	}

	for {
		if checkIfPlannedStorageChangesSuccessfullyApplied(ctx, service, plan) {
			return diags
		}

		if time.Now().Unix()-startTime > timeout {
			diags.AddError("Timeout for storage controller change expired", fmt.Sprintf("Timeout of %d s has been reached", timeout))
			return diags
		}

		time.Sleep(5 * time.Second)
	}
}

func applyStorageControllerProperties(ctx context.Context, api *gofish.APIClient, plan *models.StorageResourceModel) (diags diag.Diagnostics) {
	storage, err := getSystemStorageFromSerialNumber(api.Service, plan.StorageControllerSN.ValueString())
	if err != nil {
		diags.AddError("Requested storage serial does not match to any installed controller serial.", err.Error())
		return diags
	}

	tflog.Info(ctx, "Serial number", map[string]interface{}{
		"serial": plan.StorageControllerSN.ValueString(),
	})

	isFsas, err := IsFsasCheck(ctx, api)
	if err != nil {
		diags.AddError("Server vendor verification failed", err.Error())
		return diags
	}

	payload, anyValue := convertPlanToPayload(isFsas, *plan)

	if !anyValue {
		diags.AddError("Payload created out of defined plan will be empty.",
			"Declare at least one property which is expected to be set")
		return diags
	}

	startTime := time.Now().Unix()
	timeout := plan.JobTimeout.ValueInt64()
	taskLocation, err := patchStorageEndpoint(ctx, api.Service, storage.ODataID, payload)
	if err != nil {
		diags.AddError("Error during PATCH to storage controller.", err.Error())
		return diags
	}

	if time.Now().Unix()-startTime > timeout {
		diags.AddError("Error while waiting for resource update.", fmt.Sprintf("Timeout of %d s has been reached", timeout))
		return diags
	}

	diags = waitUntilStorageChangesApplied(ctx, api.Service, taskLocation, *plan, startTime, isFsas, timeout)
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
	state.BiosContinueOnError = types.StringValue(getOemStorage(storageConfig.StorageControllers[0].Oem).BiosContinueOnError)
	state.PatrolRead = types.StringValue(getOemStorage(storageConfig.StorageControllers[0].Oem).PatrolRead)
	state.MDCScheduleMode = types.StringValue(getOemStorage(storageConfig.StorageControllers[0].Oem).MDCScheduleMode)
	state.CoercionMode = types.StringValue(getOemStorage(storageConfig.StorageControllers[0].Oem).CoercionMode)

	if getOemStorage(storageConfig.StorageControllers[0].Oem).BiosStatusEnabled != nil {
		state.BiosStatusEnabled = types.BoolValue(*(getOemStorage(storageConfig.StorageControllers[0].Oem).BiosStatusEnabled))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).PatrolReadRatePercent != nil {
		state.PatrolReadRate = types.Int64Value(*(getOemStorage(storageConfig.StorageControllers[0].Oem).PatrolReadRatePercent))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).PatrolReadRecoverySupport != nil {
		state.PatrolReadRecoverySupport = types.BoolValue(*(getOemStorage(storageConfig.StorageControllers[0].Oem).PatrolReadRecoverySupport))
	} else {
		state.PatrolReadRecoverySupport = types.BoolValue(false)
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).BGIRate != nil {
		state.BGIRate = types.Int64Value(*(getOemStorage(storageConfig.StorageControllers[0].Oem).BGIRate))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).MDCRate != nil {
		state.MDCRate = types.Int64Value(*(getOemStorage(storageConfig.StorageControllers[0].Oem).MDCRate))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).RebuildRate != nil {
		state.RebuildRate = types.Int64Value(*(getOemStorage(storageConfig.StorageControllers[0].Oem).RebuildRate))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).MigrationRate != nil {
		state.MigrationRate = types.Int64Value(*(getOemStorage(storageConfig.StorageControllers[0].Oem).MigrationRate))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).SpindownDelay != nil {
		state.SpindownDelay = types.Int64Value(*(getOemStorage(storageConfig.StorageControllers[0].Oem).SpindownDelay))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).SpindownDelay != nil {
		state.SpinupDelay = types.Int64Value(*(getOemStorage(storageConfig.StorageControllers[0].Oem).SpindownDelay))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).SpindownUnconfiguredDrive != nil {
		state.SpindownUnconfDrive = types.BoolValue(*(getOemStorage(storageConfig.StorageControllers[0].Oem).SpindownUnconfiguredDrive))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).SpindownHotspare != nil {
		state.SpindownHotspare = types.BoolValue(*(getOemStorage(storageConfig.StorageControllers[0].Oem).SpindownHotspare))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).AutoRebuild != nil {
		state.AutoRebuild = types.BoolValue(*(getOemStorage(storageConfig.StorageControllers[0].Oem).AutoRebuild))
	}

	if getOemStorage(storageConfig.StorageControllers[0].Oem).MDCAbortOnError != nil {
		state.MDCAbortOnError = types.BoolValue(*(getOemStorage(storageConfig.StorageControllers[0].Oem).MDCAbortOnError))
	} else {
		state.MDCAbortOnError = types.BoolValue(false)
	}
	/*
				if storageConfig.StorageControllers[0].Oem.OemFujitsu.CopybackSupport != nil {
		    		state.CopybackSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.OemFujitsu.CopybackSupport)
		        }

				if storageConfig.StorageControllers[0].Oem.OemFujitsu.CopybackOnSmartErrorSupport != nil {
		    		state.CopybackOnSmartErrorSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.OemFujitsu.CopybackOnSmartErrorSupport)
		        }

				if storageConfig.StorageControllers[0].Oem.OemFujitsu.CopybackOnSSDSmartErrorSupport != nil {
		    		state.CopybackOnSSDSmartErrorSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.OemFujitsu.CopybackOnSSDSmartErrorSupport)
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
