package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BiosResourceModel struct {
	Id              types.String    `tfsdk:"id"`
	RedfishServer   []RedfishServer `tfsdk:"server"`
	Attributes      types.Map       `tfsdk:"attributes"`
	SystemResetType types.String    `tfsdk:"system_reset_type"`
	JobTimeout      types.Int64     `tfsdk:"job_timeout"`
}
