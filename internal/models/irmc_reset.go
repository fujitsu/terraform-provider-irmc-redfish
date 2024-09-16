package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IrmcResetResourceModel describes the resource data model.
type IrmcResetResourceModel struct {
	Id            types.String    `tfsdk:"id"`
	RedfishServer []RedfishServer `tfsdk:"server"`
}
