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

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	tkpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
			MarkdownDescription: "Endpoint of storage controller represented by serial number.",
			Description:         "Endpoint of storage controller represented by serial number.",
		},
		"job_timeout": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Job timeout in seconds.",
			Description:         "Job timeout in seconds.",
			Default:             int64default.StaticInt64(180),
		},
		"storage_controller_serial_number": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Serial number of storage controller.",
			Description:         "Serial number of storage controller.",
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
					"Enabled",
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
			Description:         "Spindown hotspare enabled.",
		},
		"patrol_read_recovery_support": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Patrol read recovery support enabled.",
			Description:         "Patrol read recovery support enabled.",
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
		/*
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
		*/
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

func (r *StorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-storage: create starts")

	var plan models.StorageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-storage"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	diags = applyStorageControllerProperties(ctx, api, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, diags = readStorageControllerSettingsToState(api.Service, &plan.StorageSettings)
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

	odataid, diags := readStorageControllerSettingsToState(api.Service, &state.StorageSettings)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Id = types.StringValue(odataid)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-storage: read ends")
}

func (r *StorageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-storage: update starts")

	var plan models.StorageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-storage"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	diags = applyStorageControllerProperties(ctx, api, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, diags = readStorageControllerSettingsToState(api.Service, &plan.StorageSettings)
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

type StorageImportConfig struct {
	ServerConfig
	SN string `json:"storage_controller_serial_number"`
}

func (r *StorageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "resource-storage: import starts")

	var config StorageImportConfig
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, tkpath.Root("storage_controller_serial_number"), config.SN)...)

	tflog.Info(ctx, "resource-storage: import ends")
}
