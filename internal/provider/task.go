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
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

// IsTaskFinished returns information whether task state
// has been mapped to task finished state and the information
// is returned as boolean.
func IsTaskFinished(state redfish.TaskState) bool {
	switch state {
	case redfish.CompletedTaskState, redfish.ExceptionTaskState, redfish.CancelledTaskState, redfish.KilledTaskState:
		fallthrough
	case redfish.InterruptedTaskState, redfish.SuspendedTaskState:
		return true
	default:
		break
	}
	return false
}

// IsTaskFinishedSuccessfully returns information whether task state
// has been mapped to task finished successfully or not and the information
// is returned as boolean.
func IsTaskFinishedSuccessfully(state redfish.TaskState) bool {
	switch state {
	case redfish.CompletedTaskState:
		return true
	default:
		return false
	}
}

// FetchRedfishTaskLog tries to fetch logs of task pointed by location
// from system accessed by service. If logs content could not be accessed
// diags is filled with reason.
func FetchRedfishTaskLog(service *gofish.Service, location string) (logs []byte, diags diag.Diagnostics) {
	task_log_endpoint := location + "/Oem/ts_fujitsu/Logs"
	res, err := service.GetClient().Get(task_log_endpoint)
	if err != nil {
		diags.AddError("Error while reading task log endpoint", err.Error())
		return nil, diags
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			diags.AddError("Error while reading task logs", err.Error())
			return nil, diags
		}

		return bodyBytes, diags
	} else {
		diags.AddError("Error while reading task logs", "Endpoint returned non 200 code")
		return nil, diags
	}
}

// WaitForRedfishTaskEnd checks in loop until task pointed by location on service
// will report finished state or operation will timeout (maximum time pointed by timeout_s).
// If task has been finished with success, status is returned as true. If loop has timed or
// information about task could not be retrieved, status will be returned as false with error
// pointing to reason.
func WaitForRedfishTaskEnd(ctx context.Context, service *gofish.Service, location string, timeout_s int64) (bool, error) {
	start_time := time.Now().Unix()
	for {
		task, err := redfish.GetTask(service.GetClient(), location)
		if err != nil {
			return false, fmt.Errorf("Error during task %s retrieval %s", location, err.Error())
		}

		tflog.Trace(ctx, "Task details", map[string]interface{}{
			"location": location,
			"state":    task.TaskState,
		})

		if IsTaskFinished(task.TaskState) {
			if IsTaskFinishedSuccessfully(task.TaskState) {
				return true, nil
			}

			return false, fmt.Errorf("Task finished with TaskState %s", task.TaskState)
		} else {
			time.Sleep(5 * time.Second)
		}

		if time.Now().Unix()-start_time > timeout_s {
			return false, fmt.Errorf("Task has not finished within given timeout %d", timeout_s)
		}
	}
}
