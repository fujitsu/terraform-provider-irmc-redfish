/*
Copyright (c) 2024 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

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

// VirtualMediaResourceModel describes the resource data model.
type BootSourceOverrideResourceModel struct {
	Id                        types.String    `tfsdk:"id"`
	RedfishServer             []RedfishServer `tfsdk:"server"`
	BootSourceOverrideTarget  types.String    `tfsdk:"boot_source_override_target"`
	BootSourceOverrideEnabled types.String    `tfsdk:"boot_source_override_enabled"`
	SystemResetType           types.String    `tfsdk:"system_reset_type"`
	JobTimeout                types.Int64     `tfsdk:"job_timeout"`
}
