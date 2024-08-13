package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VirtualMediaResourceModel describes the resource data model.
type BootOrderResourceModel struct {
	Id                   types.String `tfsdk:"id"`
    RedfishServer        []RedfishServer `tfsdk:"server"`
	BootOrder            types.List `tfsdk:"boot_order"`
}
