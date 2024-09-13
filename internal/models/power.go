package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PowerResourceModel describes the resource data model.
type PowerResourceModel struct {
	Id              types.String    `tfsdk:"id"`
	RedfishServer   []RedfishServer `tfsdk:"server"`
	HostPowerAction types.String    `tfsdk:"host_power_action"`
	MaxWaitTime     types.Int64     `tfsdk:"max_wait_time"`
	PowerState      types.String    `tfsdk:"power_state"`
}
