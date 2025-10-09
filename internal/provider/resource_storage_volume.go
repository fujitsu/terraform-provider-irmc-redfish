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

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &StorageVolumeResource{}
var _ resource.ResourceWithImportState = &StorageVolumeResource{}

func NewStorageVolumeResource() resource.Resource {
	return &StorageVolumeResource{}
}

// StorageVolumeResource defines the resource implementation.
type StorageVolumeResource struct {
	p *IrmcProvider
}

const (
	STORAGE_COLLECTION_ENDPOINT        = "/redfish/v1/Systems/0/Storage"
	STORAGE_VOLUME_RESOURCE_NAME       = "resource-storage_volume"
	STORAGE_VOLUME_JOB_DEFAULT_TIMEOUT = 300
)

type storageVolumeEndpoints struct {
	storageRaidCapabilitiesSuffix string
}

func (r *StorageVolumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + storageVolumeName
}

func StorageVolumeSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			Description:         "Id of handled volume",
			MarkdownDescription: "Id of handled volume",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"job_timeout": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Job timeout in seconds.",
			Description:         "Job timeout in seconds.",
			Default:             int64default.StaticInt64(STORAGE_VOLUME_JOB_DEFAULT_TIMEOUT),
		},
		"storage_controller_serial_number": schema.StringAttribute{
			Required:            true,
			Description:         "Serial number of storage controller.",
			MarkdownDescription: "Serial number of storage controller.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"raid_type": schema.StringAttribute{
			Required:            true,
			Description:         "RAID volume type depending on controller itself",
			MarkdownDescription: "RAID volume type depending on controller itself",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"RAID0",
					"RAID1",
					"RAID1E",
					"RAID10",
					"RAID5",
					"RAID50",
					"RAID6",
					"RAID60",
				}...),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"physical_drives": schema.ListAttribute{
			Required:            true,
			Description:         "List of slot locations of disks used for volume creation.",
			MarkdownDescription: "List of slot locations of disks used for volume creation.",
			ElementType:         types.StringType,
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
			},
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
		// Usually if volume is created, size of the volume is not exactly
		// the same as requested due to controller (values in bytes can be rounded).
		// For that reason semantic equality logic is required here.
		"capacity_bytes": schema.Int64Attribute{
			CustomType:          models.CapacityByteType{},
			Description:         "Volume capacity in bytes.",
			MarkdownDescription: "Volume capacity in bytes. If not specified during creation, volume will have maximum size calculated from chosen disks.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"name": schema.StringAttribute{
			Computed:            true,
			Optional:            true,
			Description:         "Volume name.",
			MarkdownDescription: "Volume name.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
				stringvalidator.LengthAtMost(15),
			},
		},
		"init_mode": schema.StringAttribute{
			Optional:            true,
			Description:         "Initialize mode for new volume.",
			MarkdownDescription: "Initialize mode for new volume.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"None",
					"Fast",
					"Normal",
				}...),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"optimum_io_size_bytes": schema.Int64Attribute{
			Description:         "Optimum IO size bytes.",
			MarkdownDescription: "Optimum IO size bytes.",
			Required:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		"read_mode": schema.SingleNestedAttribute{
			Attributes: map[string]schema.Attribute{
				"requested": schema.StringAttribute{
					Optional:            true,
					Description:         "Requested read mode of a created volume.",
					MarkdownDescription: "Requested read mode of a created volume.",
					Validators: []validator.String{
						stringvalidator.OneOf([]string{
							"Adaptive",
							"NoReadAhead",
							"ReadAhead",
						}...),
					},
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplaceIfConfigured(),
					},
				},
				"actual": schema.StringAttribute{
					Computed:            true,
					Description:         "Actual read mode of a created volume.",
					MarkdownDescription: "Actual read mode of a created volume.",
				},
			},
			Optional: true,
		},
		"write_mode": schema.SingleNestedAttribute{
			Attributes: map[string]schema.Attribute{
				"requested": schema.StringAttribute{
					Optional:            true,
					Description:         "Requested write mode of a created volume.",
					MarkdownDescription: "Requested Write mode of a created volume.",
					Validators: []validator.String{
						stringvalidator.OneOf([]string{
							"WriteBack",
							"AlwaysWriteBack",
							"WriteThrough",
						}...),
					},
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplaceIfConfigured(),
					},
				},
				"actual": schema.StringAttribute{
					Computed:            true,
					Description:         "Actual write mode of a created volume.",
					MarkdownDescription: "Actual Write mode of a created volume.",
				},
			},
			Optional: true,
		},
		"drive_cache_mode": schema.StringAttribute{
			Optional:            true,
			Description:         "Drive cache mode of volume.",
			MarkdownDescription: "Drive cache mode of volume.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Enabled",
					"Disabled",
					"Unchanged",
				}...),
			},
			Computed: true,
		},
	}
}

func (r *StorageVolumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "This resource is used to manipulate (Create, Read, Delete, Update and Import) logical volumes of iRMC system",
		MarkdownDescription: "This resource is used to manipulate (Create, Read, Delete, Update and Import) logical volumes of iRMC system",
		Attributes:          StorageVolumeSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *StorageVolumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StorageVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "storage-volume: create starts")

	// Read Terraform plan data into the model
	var plan models.StorageVolumeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	mutexPool.Lock(ctx, endpoint, STORAGE_VOLUME_RESOURCE_NAME)
	defer mutexPool.Unlock(ctx, endpoint, STORAGE_VOLUME_RESOURCE_NAME)

	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
		return
	}

	defer api.Logout()
	isFsas, err := IsFsasCheck(ctx, api)

	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}
	var state models.StorageVolumeResourceModel
	beRemoved, diags := createStorageVolume(ctx, api.Service, plan, &state, isFsas)
	if beRemoved {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-storage-volume: create ends")
}

func (r *StorageVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-storage-volume: read starts")

	// Read Terraform prior state data into the model
	var state models.StorageVolumeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
		return
	}

	defer api.Logout()

	isFsas, err := IsFsasCheck(ctx, api)

	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}

	validStorageEndpoint, err := getValidStorageEndpointFromSerial(api.Service, state.StorageControllerSN.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get valid storage id", err.Error())
		return
	}

	if updateVolumeODataId(validStorageEndpoint, &state) {
		tflog.Info(ctx, "resource-storage-volume: state controller changed its Id")
	} else {
		tflog.Info(ctx, "resource-storage-volume: storage controller has stable id")
	}

	volume, diags, _ := doesVolumeStillExist(api.Service, state.Id.ValueString())
	if volume == nil {
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = readStorageVolumeToState(volume, state.StorageControllerSN.ValueString(), &state, isFsas)
	resp.Diagnostics.Append(diags...)

	if diags.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-storage-volume: read ends")
}

func (r *StorageVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-storage-volume: update starts")

	// Read Terraform state and plan data into the model
	var state models.StorageVolumeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan models.StorageVolumeResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	mutexPool.Lock(ctx, endpoint, STORAGE_VOLUME_RESOURCE_NAME)
	defer mutexPool.Unlock(ctx, endpoint, STORAGE_VOLUME_RESOURCE_NAME)

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
		return
	}

	defer api.Logout()

	isFsas, err := IsFsasCheck(ctx, api)

	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}

	beRemoved, diags := updateStorageVolume(ctx, api.Service, plan, &state, isFsas)
	if beRemoved {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-storage-volume: update ends")
}

func (r *StorageVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-storage-volume: delete starts")

	// Read Terraform prior state data into the model
	var state models.StorageVolumeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = state.RedfishServer[0].Endpoint.ValueString()
	mutexPool.Lock(ctx, endpoint, STORAGE_VOLUME_RESOURCE_NAME)
	defer mutexPool.Unlock(ctx, endpoint, STORAGE_VOLUME_RESOURCE_NAME)

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
		return
	}

	defer api.Logout()

	isFsas, err := IsFsasCheck(ctx, api)

	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}

	// Try to delete handled volume
	diags = deleteStorageVolume(ctx, api.Service, state.Id.ValueString(), isFsas)
	resp.Diagnostics.Append(diags...)

	if diags.HasError() {
		return
	}

	tflog.Info(ctx, "resource-storage-volume: delete ends")
}

type StorageVolumeImportConfig struct {
	ServerConfig
	ID string `json:"id"`
}

func (r *StorageVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "resource-storage-volume: import starts")

	var config StorageVolumeImportConfig
	err := json.Unmarshal([]byte(req.ID), &config)
	if err != nil {
		resp.Diagnostics.AddError("Could not import configuration", err.Error())
		return
	}

	server := models.RedfishServer{
		User:        types.StringValue(config.Username),
		Password:    types.StringValue(config.Password),
		Endpoint:    types.StringValue(config.Endpoint),
		SslInsecure: types.BoolValue(config.SslInsecure),
	}

	// no need to read current configuration since terraform will call Read() once
	// import procedure will be successfully finished

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), config.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), []models.RedfishServer{server})...)

	tflog.Info(ctx, "resource-storage-volume: import ends")
}

func getStorageCommonEndpoints(isFsas bool) storageVolumeEndpoints {
	if isFsas {
		return storageVolumeEndpoints{
			storageRaidCapabilitiesSuffix: fmt.Sprintf("/Oem/%s/RAIDCapabilities", FSAS),
		}
	} else {
		return storageVolumeEndpoints{
			storageRaidCapabilitiesSuffix: fmt.Sprintf("/Oem/%s/RAIDCapabilities", TS_FUJITSU),
		}
	}
}
