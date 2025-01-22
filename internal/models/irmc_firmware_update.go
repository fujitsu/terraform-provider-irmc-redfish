/*
Copyright (c) 2025 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

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

package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IrmcFirmwareUpdateResourceModel describes the resource data model.
type IrmcFirmwareUpdateResourceModel struct {
	Id                   types.String    `tfsdk:"id"`
	RedfishServer        []RedfishServer `tfsdk:"server"`
	UpdateType           types.String    `tfsdk:"update_type"`
	IRMCPathToBinary     types.String    `tfsdk:"irmc_path_to_binary"`
	TftpServerAddr       types.String    `tfsdk:"tftp_server_addr"`
	TftpUpdateFile       types.String    `tfsdk:"tftp_update_file"`
	IRMCFlashSelector    types.String    `tfsdk:"irmc_flash_selector"`
	IRMCBootSelector     types.String    `tfsdk:"irmc_boot_selector"`
	UpdateTimeout        types.Int64     `tfsdk:"update_timeout"`
	ResetIrmcAfterUpdate types.Bool      `tfsdk:"reset_irmc_after_update"`
}
