package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VirtualMediaResourceModel describes the resource data model.
type BootSourceOverrideResourceModel struct {
	Id                        types.String    `tfsdk:"id"`
	RedfishServer             []RedfishServer `tfsdk:"server"`
	BootSourceOverrideTarget  types.String    `tfsdk:"boot_source_override_target"`
	BootSourceOverrideEnabled types.String    `tfsdk:"boot_source_override_enabled"`
	SystemResetType           types.String    `tfsdk:"system_reset_type"`
	JobTimeout                types.Int64     `tfsdk:"job_timeout"`
}
