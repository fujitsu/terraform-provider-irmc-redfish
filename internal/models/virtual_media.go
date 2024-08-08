package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VirtualMediaDataSource struct {
//	ID               types.String       `tfsdk:"id"`
	RedfishServer    []RedfishServer    `tfsdk:"server"`
	VirtualMediaData []VirtualMediaData `tfsdk:"virtual_media"`
}

type VirtualMediaData struct {
	ODataId types.String `tfsdk:"odata_id"`
	Id      types.String `tfsdk:"id"`
}

// VirtualMediaResourceModel describes the resource data model.
type VirtualMediaResourceModel struct {
	Id                   types.String `tfsdk:"id"`
    RedfishServer        []RedfishServer `tfsdk:"server"`
	Image                types.String `tfsdk:"image"`
    Inserted             types.Bool `tfsdk:"inserted"`
	TransferProtocolType types.String `tfsdk:"transfer_protocol_type"`
	WriteProtected       types.Bool `tfsdk:"write_protected"`
}
