package models

import (
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// VirtualMediaResourceModel describes the resource data model.
type StorageVolumeResourceModel struct {
    Id                   types.String `tfsdk:"id"`
    StorageId            types.String `tfsdk:"storage_controller_id"`
    RedfishServer        []RedfishServer `tfsdk:"server"`

    RaidType             types.String `tfsdk:"raid_type"`
    CapacityBytes        types.Int64 `tfsdk:"capacity_bytes"`
    VolumeName           types.String `tfsdk:"name"`
    InitMode             types.String `tfsdk:"init_mode"`
    PhysicalDrives       types.List `tfsdk:"physical_drives"`
    OptimumIOSizeBytes   types.Int64 `tfsdk:"optimum_io_size_bytes"`
    ReadMode             types.String `tfsdk:"read_mode"`
    WriteMode            types.String `tfsdk:"write_mode"`
//    CacheMode            types.String `tfsdk:"cache_mode"`
    DriveCacheMode       types.String `tfsdk:"drive_cache_mode"`
}

