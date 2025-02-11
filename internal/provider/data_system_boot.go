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

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &IrmcSystemBootDataSource{}

func NewSystemBootDataSource() datasource.DataSource {
	return &IrmcSystemBootDataSource{}
}

// IrmcSystemBootDataSource defines the data source implementation.
type IrmcSystemBootDataSource struct {
	p *IrmcProvider
}

func (d *IrmcSystemBootDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + systemBoot
}

func SystemBootDataSourceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "ID of the system boot resource",
		},
		"boot_order": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Boot order of the system",
		},
		"boot_source_override_enabled": schema.StringAttribute{
			Computed:    true,
			Description: "Indicates whether boot source override is enabled",
		},
		"boot_source_override_mode": schema.StringAttribute{
			Computed:    true,
			Description: "Mode of boot source override",
		},
		"boot_source_override_target": schema.StringAttribute{
			Computed:    true,
			Description: "Target of boot source override",
		},
	}
}

func (d *IrmcSystemBootDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for retrieving system boot information.",
		Attributes:          SystemBootDataSourceSchema(),
		Blocks:              RedfishServerDatasourceBlockMap(),
	}
}

func (d *IrmcSystemBootDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	p, ok := req.ProviderData.(*IrmcProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.p = p
}

func (d *IrmcSystemBootDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "data-system-boot: read starts")

	var data models.SystemBootDataSource
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Error parsing configuration data")
		return
	}

	api, err := ConnectTargetSystem(d.p, &data.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connection Error", err.Error())
		return
	}
	defer api.Logout()

	system, err := GetSystemResource(api.Service)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching System Resource", err.Error())
		return
	}

	if system == nil {
		resp.Diagnostics.AddError("System Not Found", "No matching system resource found")
		return
	}

	boot := system.Boot

	rBios, err := system.Bios()
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching BIOS Resource", err.Error())
		return
	}

	bootOrder := []attr.Value{}

	if len(rBios.Attributes) > 0 {
		if currentBootConfigOrder, exists := rBios.Attributes[PERSISTENT_BOOT_ORDER_KEY]; exists {
			bootOrderStr, _ := json.Marshal(currentBootConfigOrder)
			var bootOrderList []BootEntry
			if err := json.Unmarshal(bootOrderStr, &bootOrderList); err != nil {
				resp.Diagnostics.AddError("Error Unmarshalling PersistentBootConfigOrder", err.Error())
				return
			}

			for _, item := range bootOrderList {
				bootOrder = append(bootOrder, types.StringValue(item.StructuredBootString))
			}
		}
	}

	data.ID = types.StringValue(system.ODataID)
	data.BootOrder = types.ListValueMust(types.StringType, bootOrder)
	data.BootSourceOverrideEnabled = types.StringValue(string(boot.BootSourceOverrideEnabled))
	data.BootSourceOverrideMode = types.StringValue(string(boot.BootSourceOverrideMode))
	data.BootSourceOverrideTarget = types.StringValue(string(boot.BootSourceOverrideTarget))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "data-system-boot: read ends")
}
