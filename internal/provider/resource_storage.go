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

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tkpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &StorageResource{}
var _ resource.ResourceWithImportState = &StorageResource{}

func NewStorageResource() resource.Resource {
	return &StorageResource{}
}

// StorageResource defines the resource implementation.
type StorageResource struct {
	p *IrmcProvider
}

func (r *StorageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + storageName
}

func StorageControllerSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of BIOS settings resource on iRMC.",
			Description:         "ID of BIOS settings resource on iRMC.",
		},
		"job_timeout": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Job timeout in seconds.",
			Description:         "Job timeout in seconds.",
		},
		"storage_controller_serial_number": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Serial number of storage_controller.",
			Description:         "Serial number of storage_controller.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"bios_continue_on_error": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "BIOS continue on error.",
			Description:         "BIOS continue on error.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"StopOnErrors",
					"PauseOnErrors",
					"IgnoreErrors",
					"SafeModeOnErrors",
				}...),
			},
		},
		"bios_status": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "BIOS status.",
			Description:         "BIOS status.",
		},
		"patrol_read": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Patrol read.",
			Description:         "Patrol read.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Automatic",
					"Disabled",
					"Manual",
				}...),
			},
		},
		"patrol_read_rate": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Patrol read rate percent.",
			Description:         "Patrol read rate percent.",
			Validators: []validator.Int64{
				int64validator.Between(0, 100),
			},
		},
		"bgi_rate": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "BGI rate percent.",
			Description:         "BGI rate percent.",
			Validators: []validator.Int64{
				int64validator.Between(0, 100),
			},
		},
		"mdc_rate": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "MDC rate percent.",
			Description:         "MDC rate percent.",
			Validators: []validator.Int64{
				int64validator.Between(0, 100),
			},
		},
		"rebuild_rate": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Rebuild rate percent.",
			Description:         "Rebuild rate percent.",
			Validators: []validator.Int64{
				int64validator.Between(0, 100),
			},
		},
		"migration_rate": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Migration rate percent.",
			Description:         "Migration rate percent.",
			Validators: []validator.Int64{
				int64validator.Between(0, 100),
			},
		},
		"spindown_delay": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Spindown delay.",
			Description:         "Spindown delay.",
			Validators: []validator.Int64{
				int64validator.Between(30, 1440),
			},
		},
		"spinup_delay": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Spinup delay.",
			Description:         "Spinup delay.",
			Validators: []validator.Int64{
				int64validator.Between(0, 6),
			},
		},
		"spindown_unconfigured_drive_enabled": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Spindown unconfigured drive enabled.",
			Description:         "Spindown unconfigured drive enabled.",
		},
		"spindown_hotspare_enabled": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Spindown hotspare enabled.",
			Description:         "Spindown hotspare.",
		},
		"patrol_read_recovery_support": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Patrol read recovery support.",
			Description:         "Patrol read recovery support.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Disabled",
					"Enabled",
				}...),
			},
		},
		"mdc_schedule_mode": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "MDC schedule mode.",
			Description:         "MDC schedule mode.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Disabled",
					"Sequential",
					"Concurrent",
				}...),
			},
		},
		"mdc_abort_on_error_enabled": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "MDC abort on error enabled.",
			Description:         "MDC abort on error enabled.",
		},
		"coercion_mode": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Coercion mode.",
			Description:         "Coercion mode.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"None",
					"Coerce128MiB",
					"Coerce1GiB",
				}...),
			},
		},
		"copyback_support_enabled": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Copyback support enabled.",
			Description:         "Copyback support enabled.",
		},
		"copyback_on_smart_error_support_enabled": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Copyback on smart error support enabled.",
			Description:         "Copyback on smart error support enabled.",
		},
		"copyback_on_ssd_smart_error_support_enabled": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Copyback on SSD smart error support enabled.",
			Description:         "Copyback on SSD smart error support enabled.",
		},
		"auto_rebuild_enabled": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Auto rebuild enabled.",
			Description:         "Auto rebuild enabled.",
		},
	}
}

func (r *StorageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The resource is used to control (read, modify or import) storage controller settings on Fujitsu server equipped with iRMC controller.",
		Description:         "The resource is used to control (read, modify or import) storage controller settings on Fujitsu server equipped with iRMC controller.",
		Attributes:          StorageControllerSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *StorageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	p, ok := req.ProviderData.(*IrmcProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *IrmcProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.p = p
}

type storageControllerOem struct {
	BiosContinueOnError   string `json:"BIOSContinueOnError,omitempty"`
	BiosStatus            bool   `json:"BIOSStatus,omitempty"`
	PatrolRead            string `json:"PatrolRead,omitempty"`
	PatrolReadRatePercent int64  `json:"PatrolReadRate,omitempty"`
	BGIRate               int64  `json:"BGIRate,omitempty"`
	MDCRate               int64  `json:"MDCRate,omitempty"`
	RebuildRate           int64  `json:"RebuildRate,omitempty"`
	MigrationRate         int64  `json:"MigrationRate,omitempty"`

	SpinupDelay                    int64  `json:"SpinupDelaySec,omitempty"`
	SpindownDelay                  int64  `json:"SpindownDelayMin,omitempty"`
	SpindownUnconfiguredDrive      bool   `json:"SpindownUnconfiguredDrive,omitempty"`
	SpindownHotspare               bool   `json:"SpindownHotspare,omitempty"`
	MDCScheduleMode                string `json:"MDCScheduleMode,omitempty"`
	MDCAbortOnError                bool   `json:"MDCAbortOnError,omitempty"`
	CoercionMode                   string `json:"CoercionMode,omitempty"`
	CopybackSupport                bool   `json:"CopybackSupport,omitempty"`
	CopybackOnSmartErrorSupport    bool   `json:"CopybackOnSMARTErrSupport,omitempty"`
	CopybackOnSSDSmartErrorSupport bool   `json:"CopybackOnSSDSMARTErrSupport,omitempty"`
	AutoRebuild                    bool   `json:"AutoRebuildSupport,omitempty"`
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

func (r *StorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-storage: create starts")

	// Read Terraform plan data into the model
	var plan models.StorageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-storage"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	diags = applyStorageControllerProperties(ctx, api.Service, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-storage: create ends")
}

func (r *StorageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-storage: read starts")

	// Read Terraform prior state data into the model
	var state models.StorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	diags := readStorageControllerSettingsToState(api.Service, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-storage: read ends")
}

func (r *StorageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-storage: update starts")

	// Read Terraform plan
	var plan models.StorageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	diags = applyStorageControllerProperties(ctx, api.Service, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-storage: update ends")
}

func (r *StorageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-storage: delete starts")
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-storage: delete ends")
}

func (r *StorageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "resource-storage: import starts")

	var config CommonImportConfig
	err := json.Unmarshal([]byte(req.ID), &config)
	if err != nil {
		resp.Diagnostics.AddError("Error while unmarshalling import config", err.Error())
		return
	}

	server := models.RedfishServer{
		User:        types.StringValue(config.Username),
		Password:    types.StringValue(config.Password),
		Endpoint:    types.StringValue(config.Endpoint),
		SslInsecure: types.BoolValue(config.SslInsecure),
	}

	creds := []models.RedfishServer{server}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, tkpath.Root("server"), creds)...)

	tflog.Info(ctx, "resource-storage: import ends")
}

// TO BE REMOVED!
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

	return nil, fmt.Errorf("Requested Storage resource has not been found on list")
}

func convertPlanToPayload(plan models.StorageResourceModel) any {
	var payload Storage_Fujitsu
	var storageController StorageController_Fujitsu

	if !plan.BGIRate.IsNull() && !plan.BGIRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.BGIRate = plan.BGIRate.ValueInt64()
	}

	if !plan.MDCRate.IsNull() && !plan.MDCRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.MDCRate = plan.MDCRate.ValueInt64()
	}

	if !plan.BiosStatusEnabled.IsNull() && !plan.BiosStatusEnabled.IsUnknown() {
		storageController.Oem.Ts_fujitsu.BiosStatus = plan.BiosStatusEnabled.ValueBool()
	}

	if !plan.BiosContinueOnError.IsNull() && !plan.BiosContinueOnError.IsUnknown() {
		storageController.Oem.Ts_fujitsu.BiosContinueOnError = plan.BiosContinueOnError.ValueString()
	}

	if !plan.PatrolRead.IsNull() && !plan.PatrolRead.IsUnknown() {
		storageController.Oem.Ts_fujitsu.PatrolRead = plan.PatrolRead.ValueString()
	}

	if !plan.PatrolReadRate.IsNull() && !plan.PatrolReadRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.PatrolReadRatePercent = plan.PatrolReadRate.ValueInt64()
	}

	if !plan.RebuildRate.IsNull() && !plan.RebuildRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.RebuildRate = plan.RebuildRate.ValueInt64()
	}

	if !plan.MigrationRate.IsNull() && !plan.MigrationRate.IsUnknown() {
		storageController.Oem.Ts_fujitsu.MigrationRate = plan.MigrationRate.ValueInt64()
	}

	payload.StorageControllers = append(payload.StorageControllers, storageController)
	return payload
}

func applyPayload(ctx context.Context, service *gofish.Service, storage_endpoint string, payload any) (taskLocation string, err error) {
	tflog.Info(ctx, "Changes will be applied to controller", map[string]interface{}{
		"storage endpoint": storage_endpoint,
		"payload":          payload,
	})

	res, err := service.GetClient().Patch(storage_endpoint, payload)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PATCH request on '%s' finished with not expected status '%d'", storage_endpoint, res.StatusCode)
	}
	return "", err
}

func checkAppliedSettingsFromPlan(ctx context.Context, plan models.StorageResourceModel, current Storage_Fujitsu) (status bool) {
	status = true

	if !plan.BiosContinueOnError.IsNull() && !plan.BiosContinueOnError.IsUnknown() {
		if plan.BiosContinueOnError.ValueString() != current.StorageControllers[0].Oem.Ts_fujitsu.BiosContinueOnError {
			tflog.Info(ctx, "Value for property BIOSContinueOnError has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BiosContinueOnError.ValueString(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.BiosContinueOnError,
			})
			status = false
		}
	}

	if !plan.BiosStatusEnabled.IsNull() && !plan.BiosStatusEnabled.IsUnknown() {
		if plan.BiosStatusEnabled.ValueBool() != current.StorageControllers[0].Oem.Ts_fujitsu.BiosStatus {
			status = false
			tflog.Trace(ctx, "Value for property BIOSStatus has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BiosStatusEnabled.ValueBool(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.BiosStatus,
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
		if plan.PatrolReadRate.ValueInt64() != current.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRatePercent {
			status = false
			tflog.Info(ctx, "Value for property PatrolReadRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.PatrolReadRate.ValueInt64(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRatePercent,
			})
		}
	}

	if !plan.BGIRate.IsNull() && !plan.BGIRate.IsUnknown() {
		if plan.BGIRate.ValueInt64() != current.StorageControllers[0].Oem.Ts_fujitsu.BGIRate {
			status = false
			tflog.Info(ctx, "Value for property BGIRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.BGIRate.ValueInt64(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.BGIRate,
			})
		}
	}

	if !plan.MDCRate.IsNull() && !plan.MDCRate.IsUnknown() {
		if plan.MDCRate.ValueInt64() != current.StorageControllers[0].Oem.Ts_fujitsu.MDCRate {
			status = false
			tflog.Info(ctx, "Value for property MDCRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MDCRate.ValueInt64(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.MDCRate,
			})
		}
	}

	if !plan.RebuildRate.IsNull() && !plan.RebuildRate.IsUnknown() {
		if plan.RebuildRate.ValueInt64() != current.StorageControllers[0].Oem.Ts_fujitsu.RebuildRate {
			status = false
			tflog.Info(ctx, "Value for property RebuildRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.RebuildRate.ValueInt64(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.RebuildRate,
			})
		}
	}

	if !plan.MigrationRate.IsNull() && !plan.MigrationRate.IsUnknown() {
		if plan.MigrationRate.ValueInt64() != current.StorageControllers[0].Oem.Ts_fujitsu.MigrationRate {
			status = false
			tflog.Info(ctx, "Value for property MigrationRate has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MigrationRate.ValueInt64(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.MigrationRate,
			})
		}
	}

	if !plan.SpindownDelay.IsNull() && !plan.SpindownDelay.IsUnknown() {
		if plan.SpindownDelay.ValueInt64() != current.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay {
			status = false
			tflog.Info(ctx, "Value for property SpindownDelay has not yet reached planned value", map[string]interface{}{
				"plan":     plan.MigrationRate.ValueInt64(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay,
			})
		}
	}

	if !plan.SpinupDelay.IsNull() && !plan.SpinupDelay.IsUnknown() {
		if plan.SpinupDelay.ValueInt64() != current.StorageControllers[0].Oem.Ts_fujitsu.SpinupDelay {
			status = false
			tflog.Info(ctx, "Value for property SpinupDelay has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpinupDelay.ValueInt64(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.SpinupDelay,
			})
		}
	}

	if !plan.SpindownUnconfDrive.IsNull() && !plan.SpindownUnconfDrive.IsUnknown() {
		if plan.SpindownUnconfDrive.ValueBool() != current.StorageControllers[0].Oem.Ts_fujitsu.SpindownUnconfiguredDrive {
			status = false
			tflog.Info(ctx, "Value for property SpindownUnconfiguredDrive has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpindownUnconfDrive.ValueBool(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.SpindownUnconfiguredDrive,
			})
		}
	}

	if !plan.SpindownHotspare.IsNull() && !plan.SpindownHotspare.IsUnknown() {
		if plan.SpindownHotspare.ValueBool() != current.StorageControllers[0].Oem.Ts_fujitsu.SpindownHotspare {
			status = false
			tflog.Info(ctx, "Value for property SpindownHotspare has not yet reached planned value", map[string]interface{}{
				"plan":     plan.SpindownHotspare.ValueBool(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.SpindownHotspare,
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
		if plan.MDCAbortOnError.ValueBool() != current.StorageControllers[0].Oem.Ts_fujitsu.MDCAbortOnError {
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

	if !plan.AutoRebuild.IsNull() && !plan.AutoRebuild.IsUnknown() {
		if plan.AutoRebuild.ValueBool() != current.StorageControllers[0].Oem.Ts_fujitsu.AutoRebuild {
			status = false
			tflog.Info(ctx, "Value for property AutoRebuild has not yet reached planned value", map[string]interface{}{
				"plan":     plan.AutoRebuild.ValueBool(),
				"reported": current.StorageControllers[0].Oem.Ts_fujitsu.AutoRebuild,
			})
		}
	}

	return status
}

func checkIfPlannedChangesSuccessfullyApplied(ctx context.Context, service *gofish.Service, plan models.StorageResourceModel) bool {
	var storageResource Storage_Fujitsu
	err := readStorageControllerSettings(service, plan.StorageControllerSN.ValueString(), &storageResource)
	if err != nil {
		return false
	}

	return checkAppliedSettingsFromPlan(ctx, plan, storageResource)
}

func waitUntilChangesApplied(ctx context.Context, service *gofish.Service, taskLocation string, plan models.StorageResourceModel, startTime int64, timeout int64) (status bool, err error) {
	for {
		if len(taskLocation) == 0 {
			if checkIfPlannedChangesSuccessfullyApplied(ctx, service, plan) {
				return true, err
			}
		}
		// TODO: no support for task approach

		if time.Now().Unix()-startTime > timeout {
			return false, fmt.Errorf("Timeout of %d s has been reached", timeout)
		}

		time.Sleep(5 * time.Second)
	}
}

func applyStorageControllerProperties(ctx context.Context, service *gofish.Service, plan *models.StorageResourceModel) (diags diag.Diagnostics) {
	payload := convertPlanToPayload(*plan)
	storage, err := getSystemStorageFromSerialNumber(service, plan.StorageControllerSN.ValueString())
	if err != nil {
		diags.AddError("Error during storage SN to storage resource.", err.Error())
		return diags
	}

	startTime := time.Now().Unix()
	taskLocation, err := applyPayload(ctx, service, storage.ODataID, payload)
	if err != nil {
		diags.AddError("Error during PATCH to storage controller.", err.Error())
		return diags
	}

	_, err = waitUntilChangesApplied(ctx, service, taskLocation, *plan, startTime, plan.JobTimeout.ValueInt64())
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

func copyStorageConfigIntoModel(storageConfig Storage_Fujitsu, state *models.StorageResourceModel) {
	state.BiosContinueOnError = types.StringValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.BiosContinueOnError)
	state.BiosStatusEnabled = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.BiosStatus)
	state.PatrolRead = types.StringValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.PatrolRead)
	state.PatrolReadRate = types.Int64Value(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.PatrolReadRatePercent)
	state.BGIRate = types.Int64Value(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.BGIRate)
	state.MDCRate = types.Int64Value(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MDCRate)
	state.RebuildRate = types.Int64Value(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.RebuildRate)
	state.MigrationRate = types.Int64Value(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MigrationRate)

	state.SpindownDelay = types.Int64Value(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay)
	state.SpinupDelay = types.Int64Value(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownDelay)
	state.SpindownUnconfDrive = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownUnconfiguredDrive)
	state.SpindownHotspare = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.SpindownHotspare)
	state.MDCScheduleMode = types.StringValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MDCScheduleMode)
	state.MDCAbortOnError = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.MDCAbortOnError)
	state.CoercionMode = types.StringValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CoercionMode)
	state.CopybackSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackSupport)
	state.CopybackOnSmartErrorSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSmartErrorSupport)
	state.CopybackOnSSDSmartErrorSupport = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.CopybackOnSSDSmartErrorSupport)
	state.AutoRebuild = types.BoolValue(storageConfig.StorageControllers[0].Oem.Ts_fujitsu.AutoRebuild)
}

func readStorageControllerSettings(service *gofish.Service, serialNumber string, storageResource *Storage_Fujitsu) (err error) {
	storage, err := getSystemStorageFromSerialNumber(service, serialNumber)
	if err != nil {
		return err
	}

	err = getParsedStorageResource(service, storage.ODataID, storageResource)
	if err != nil {
		return err
	}

	return nil
}

func readStorageControllerSettingsToState(service *gofish.Service, state *models.StorageResourceModel) (diags diag.Diagnostics) {
	var storageResource Storage_Fujitsu
	err := readStorageControllerSettings(service, state.StorageControllerSN.ValueString(), &storageResource)
	if err != nil {
		diags.AddError("Could not obtain storage resource settings", err.Error())
		return diags
	}

	copyStorageConfigIntoModel(storageResource, state)
	return diags
}
