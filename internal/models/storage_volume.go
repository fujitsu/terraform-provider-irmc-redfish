package models

import (
	"context"
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type CapacityByteValue struct {
	basetypes.Int64Value
}

var _ basetypes.Int64Valuable = CapacityByteValue{}
var _ basetypes.Int64ValuableWithSemanticEquals = CapacityByteValue{}
var _ basetypes.Int64Typable = CapacityByteType{}

type CapacityByteType struct {
	basetypes.Int64Type
}

func (t CapacityByteType) Equal(o attr.Type) bool {
	return true
}

func (t CapacityByteType) String() string {
	return "CapacityByteType"
}

func (t CapacityByteType) ValueFromInt64(ctx context.Context, in basetypes.Int64Value) (basetypes.Int64Valuable, diag.Diagnostics) {
	value := CapacityByteValue{
		Int64Value: in,
	}

	return value, nil
}

func (t CapacityByteType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.Int64Type.ValueFromTerraform(ctx, in)

	if err != nil {
		return nil, err
	}

	intValue, ok := attrValue.(basetypes.Int64Value)

	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	intValuable, diags := t.ValueFromInt64(ctx, intValue)

	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting Int64Value to IntValuable: %v", diags)
	}

	return intValuable, nil
}

func (v CapacityByteType) ValueType(ctx context.Context) attr.Value {
	return CapacityByteValue{}
}

func (v CapacityByteValue) Int64SemanticEquals(_ context.Context, newValueable basetypes.Int64Valuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, ok := newValueable.(CapacityByteValue)
	if !ok {
		diags.AddError("Semantics equality check error", "")
		return false, diags
	}

	diff := math.Abs(float64(v.Int64Value.ValueInt64() - newValue.ValueInt64()))
	if diff < 500000000 {
		return true, diags
	}

	var buff string = fmt.Sprintf("Current volume capacity differs too much vs requested value (%f bytes while allowed 500000000 bytes)", diff)
	diags.AddError("Int64SemanticsEquals", buff)
	return false, diags
}

func (v CapacityByteValue) Equal(o attr.Value) bool {
	newValue, ok := o.(CapacityByteValue)
	if !ok {
		return false
	}

	return v.Int64Value.Equal(newValue.Int64Value)
}

func (v CapacityByteValue) Type(ctx context.Context) attr.Type {
	return CapacityByteType{}
}

type StorageVolumeDynamicParam struct {
    Requested types.String `tfsdk:"requested"`
    Actual types.String `tfsdk:"actual"`
}

// StorageVolumeResourceModel describes the resource data model.
type StorageVolumeResourceModel struct {
	Id                  types.String    `tfsdk:"id"`
	StorageControllerSN types.String    `tfsdk:"storage_controller_serial_number"`
	RedfishServer       []RedfishServer `tfsdk:"server"`
    JobTimeout    types.Int64     `tfsdk:"job_timeout"`

	RaidType           types.String      `tfsdk:"raid_type"`
	CapacityBytes      CapacityByteValue `tfsdk:"capacity_bytes"`
	VolumeName         types.String      `tfsdk:"name"`
	InitMode           types.String      `tfsdk:"init_mode"`
	PhysicalDrives     types.List        `tfsdk:"physical_drives"`
	OptimumIOSizeBytes types.Int64       `tfsdk:"optimum_io_size_bytes"`
	ReadMode           StorageVolumeDynamicParam      `tfsdk:"read_mode"`
	WriteMode          StorageVolumeDynamicParam      `tfsdk:"write_mode"`
	DriveCacheMode     types.String      `tfsdk:"drive_cache_mode"`
}
