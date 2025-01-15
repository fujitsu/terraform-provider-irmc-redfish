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
	"fmt"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &StorageDataSource{}

func NewStorageDataSource() datasource.DataSource {
	return &StorageDataSource{}
}

// StorageDataSource defines the data source implementation.
type StorageDataSource struct {
	p *IrmcProvider
}

func (d *StorageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + storageName
}

func StorageDataSourceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of BIOS settings resource on iRMC.",
			Description:         "ID of BIOS settings resource on iRMC.",
		},
		"storage_controller_serial_number": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Serial number of storage controller.",
			Description:         "Serial number of storage controller.",
		},
		"bios_continue_on_error": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "BIOS continue on error.",
			Description:         "BIOS continue on error.",
		},
		"bios_status": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "BIOS status.",
			Description:         "BIOS status.",
		},
		"patrol_read": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Patrol read.",
			Description:         "Patrol read.",
		},
		"patrol_read_rate": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Patrol read rate percent.",
			Description:         "Patrol read rate percent.",
		},
		"bgi_rate": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "BGI rate percent.",
			Description:         "BGI rate percent.",
		},
		"mdc_rate": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "MDC rate percent.",
			Description:         "MDC rate percent.",
		},
		"rebuild_rate": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Rebuild rate percent.",
			Description:         "Rebuild rate percent.",
		},
		"migration_rate": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Migration rate percent.",
			Description:         "Migration rate percent.",
		},
		"spindown_delay": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Spindown delay.",
			Description:         "Spindown delay.",
		},
		"spinup_delay": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Spinup delay.",
			Description:         "Spinup delay.",
		},
		"spindown_unconfigured_drive_enabled": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Spindown unconfigured drive enabled.",
			Description:         "Spindown unconfigured drive enabled.",
		},
		"spindown_hotspare_enabled": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Spindown hotspare enabled.",
			Description:         "Spindown hotspare.",
		},
		"patrol_read_recovery_support": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Patrol read recovery support enabled.",
			Description:         "Patrol read recovery support enabled.",
		},
		"mdc_schedule_mode": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "MDC schedule mode.",
			Description:         "MDC schedule mode.",
		},
		"mdc_abort_on_error_enabled": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "MDC abort on error enabled.",
			Description:         "MDC abort on error enabled.",
		},
		"coercion_mode": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Coercion mode.",
			Description:         "Coercion mode.",
		},
		/*
			"copyback_support_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Copyback support enabled.",
				Description:         "Copyback support enabled.",
			},
			"copyback_on_smart_error_support_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Copyback on smart error support enabled.",
				Description:         "Copyback on smart error support enabled.",
			},
			"copyback_on_ssd_smart_error_support_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Copyback on SSD smart error support enabled.",
				Description:         "Copyback on SSD smart error support enabled.",
			},
		*/
		"auto_rebuild_enabled": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Auto rebuild enabled.",
			Description:         "Auto rebuild enabled.",
		},
	}
}

func (d *StorageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Storage data source",
		Attributes:          StorageDataSourceSchema(),
		Blocks:              RedfishServerDatasourceBlockMap(),
	}
}

func (d *StorageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *StorageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "data-source-storage: read starts")

	var state models.StorageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, err := ConnectTargetSystem(d.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	odataid, diags := readStorageControllerSettingsToState(api.Service, &state.StorageSettings)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Id = types.StringValue(odataid)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "data-source-storage: read ends")
}
