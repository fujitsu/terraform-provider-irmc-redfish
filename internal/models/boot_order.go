package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BootOrderResourceModel struct {
	Id              types.String    `tfsdk:"id"`
	RedfishServer   []RedfishServer `tfsdk:"server"`
	BootOrder       types.List      `tfsdk:"boot_order"`
	SystemResetType types.String    `tfsdk:"system_reset_type"`
	JobTimeout      types.Int64     `tfsdk:"job_timeout"`
}
