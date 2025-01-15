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

package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type StorageSettings struct {
	StorageControllerSN       types.String `tfsdk:"storage_controller_serial_number"`
	BiosContinueOnError       types.String `tfsdk:"bios_continue_on_error"`
	BiosStatusEnabled         types.Bool   `tfsdk:"bios_status"`
	PatrolRead                types.String `tfsdk:"patrol_read"`
	PatrolReadRate            types.Int64  `tfsdk:"patrol_read_rate"`
	PatrolReadRecoverySupport types.Bool   `tfsdk:"patrol_read_recovery_support"`
	BGIRate                   types.Int64  `tfsdk:"bgi_rate"`
	MDCRate                   types.Int64  `tfsdk:"mdc_rate"`
	RebuildRate               types.Int64  `tfsdk:"rebuild_rate"`
	MigrationRate             types.Int64  `tfsdk:"migration_rate"`
	SpindownDelay             types.Int64  `tfsdk:"spindown_delay"`
	SpinupDelay               types.Int64  `tfsdk:"spinup_delay"`
	SpindownUnconfDrive       types.Bool   `tfsdk:"spindown_unconfigured_drive_enabled"`
	SpindownHotspare          types.Bool   `tfsdk:"spindown_hotspare_enabled"`
	MDCScheduleMode           types.String `tfsdk:"mdc_schedule_mode"`
	MDCAbortOnError           types.Bool   `tfsdk:"mdc_abort_on_error_enabled"`
	CoercionMode              types.String `tfsdk:"coercion_mode"`
	/*
		CopybackSupport                types.Bool   `tfsdk:"copyback_support_enabled"`
		CopybackOnSmartErrorSupport    types.Bool   `tfsdk:"copyback_on_smart_error_support_enabled"`
		CopybackOnSSDSmartErrorSupport types.Bool   `tfsdk:"copyback_on_ssd_smart_error_support_enabled"`
	*/
	AutoRebuild types.Bool `tfsdk:"auto_rebuild_enabled"`
}

type StorageResourceModel struct {
	Id            types.String    `tfsdk:"id"`
	RedfishServer []RedfishServer `tfsdk:"server"`
	JobTimeout    types.Int64     `tfsdk:"job_timeout"`

	StorageSettings
}

type StorageDataSourceModel struct {
	Id            types.String    `tfsdk:"id"`
	RedfishServer []RedfishServer `tfsdk:"server"`

	StorageSettings
}
